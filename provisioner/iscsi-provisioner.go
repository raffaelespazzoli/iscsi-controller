package provisioner

import (
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/raffaelespazzoli/iscsi-controller/provisioner/jsonrpc2"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	//"net/rpc"
	//"net/rpc/jsonrpc"
	"sort"
)

var log = logrus.New()

type vol_createArgs struct {
	pool string
	name string
	size int64
}

type vol_destroyArgs struct {
	pool string
	name string
}

type export_createArgs struct {
	pool          string
	vol           string
	initiator_wwn string
	lun           int32
}

type export_destroyArgs struct {
	pool          string
	vol           string
	initiator_wwn string
}

type iscsiProvisioner struct {
	targetdURL    string
	initiator_wwn string
	volume_group  string
}

type export struct {
	initiator_wwn string
	lun           int32
	vol_name      string
	vol_size      int
	vol_uuid      string
	pool          string
}

type exportList []export

type result int

func NewiscsiProvisioner(url, initiator_wwn string, volume_group string) controller.Provisioner {

	initLog()

	return &iscsiProvisioner{
		targetdURL:    url,
		initiator_wwn: initiator_wwn,
		volume_group:  volume_group,
	}
}

// Provision creates a storage asset and returns a PV object representing it.
func (p *iscsiProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	log.Debugln("new provision request received for pvc: ", options.PVName)
	vol, lun, err := p.createVolume(options)
	if err != nil {
		log.Warnln(err)
		return nil, err
	}
	log.Debugln("volume created with vol and lun: ", vol, lun)

	annotations := make(map[string]string)
	annotations["volume_name"] = vol
	annotations["pool"] = options.Parameters["pool"]
	//	annotations[annExportBlock] = exportBlock
	//	annotations[annExportID] = strconv.FormatUint(uint64(exportID), 10)
	//	annotations[annProjectBlock] = projectBlock
	//	annotations[annProjectID] = strconv.FormatUint(uint64(projectID), 10)
	//	if supGroup != 0 {
	//		annotations[VolumeGidAnnotationKey] = strconv.FormatUint(supGroup, 10)
	//	}
	//	annotations[annProvisionerID] = string(p.identity)

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        options.PVName,
			Labels:      map[string]string{},
			Annotations: annotations,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				ISCSI: &v1.ISCSIVolumeSource{
					TargetPortal:   options.Parameters["targetPortal"],
					IQN:            options.Parameters["iqn"],
					ISCSIInterface: options.Parameters["iscsiInterface"],
					Lun:            lun,
					ReadOnly:       false,
					FSType:         "zfs",
				},
			},
		},
	}
	return pv, nil
}

// Delete removes the storage asset that was created by Provision represented
// by the given PV.
func (p *iscsiProvisioner) Delete(volume *v1.PersistentVolume) error {
	//vol from the annotation
	log.Debugln("volume deletion request received: ", volume.GetName())
	err := p.export_destroy(volume.Annotations["volume_name"], volume.Annotations["pool"])
	if err != nil {
		log.Warnln(err)
		return err
	}
	log.Debugln("iscsi export removed: ", volume.GetName(), volume.Annotations["volume_name"], volume.Annotations["pool"])
	err = p.vol_destroy(volume.Annotations["volume_name"], volume.Annotations["pool"])
	if err != nil {
		log.Warnln(err)
		return err
	}
	log.Debugln("logical volume removed from volume group: ", volume.GetName(), volume.Annotations["volume_name"], volume.Annotations["pool"])
	return nil
}

func initLog() {
	var err error
	log.Level, err = logrus.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		log.Fatalln(err)
	}
}

func (p *iscsiProvisioner) createVolume(options controller.VolumeOptions) (vol string, lun int32, err error) {

	size := getSize(options)
	vol = p.volume_group + "/" + p.getVolumeName(options)
	lun, err = p.getFirstAvailableLun()
	pool := options.Parameters["pool"]
	if err != nil {
		log.Warnln(err)
		return "", 0, err
	}
	log.Debugln("creating volume name, size, pool: ", vol, size, pool)
	err = p.vol_create(vol, size, pool)
	if err != nil {
		log.Warnln(err)
		return "", 0, err
	}
	log.Debugln("created volume name, size, pool: ", vol, size, pool)
	log.Debugln("exporting volume name, lun pool: ", vol, lun, pool)
	err = p.export_create(vol, lun, pool)
	if err != nil {
		log.Warnln(err)
		return "", 0, err
	}
	log.Debugln("exported volume name, lun pool: ", vol, lun, pool)
	return vol, lun, nil
}

func getSize(options controller.VolumeOptions) int64 {
	q := options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	return q.Value()
}

func (p *iscsiProvisioner) getVolumeName(options controller.VolumeOptions) string {
	return options.PVName
}

func (p *iscsiProvisioner) getFirstAvailableLun() (int32, error) {
	log.Debugln("calling export_list")
	exportList, err := p.export_list()
	if err != nil {
		log.Warnln(err)
		return -1, err
	}
	log.Debugln("export_list called")
	if len(exportList) == 255 {
		return -1, errors.New("255 luns allocated no more luns available")
	}
	lun := int32(-1)
	sort.Sort(exportList)
	for i, export := range exportList {
		if i < int(export.lun) {
			lun = int32(i)
			break
		}
	}
	if lun == -1 {
		lun = int32(len(exportList))
	}
	return lun, nil
	//return 0, nil
}

////// json rpc operations ////
func (p *iscsiProvisioner) vol_destroy(vol string, pool string) error {
	client, err := p.getConnection()
	defer client.Close()
	if err != nil {
		log.Warnln(err)
		return err
	}

	//make arguments object
	args := vol_destroyArgs{
		pool: pool,
		name: vol,
	}
	//this will store returned result
	var result result
	//call remote procedure with args
	err = client.Call("vol_destroy", args, &result)
	return err
}

func (p *iscsiProvisioner) export_destroy(vol string, pool string) error {

	client, err := p.getConnection()
	defer client.Close()
	if err != nil {
		log.Warnln(err)
		return err
	}

	//make arguments object
	args := export_destroyArgs{
		pool:          pool,
		vol:           vol,
		initiator_wwn: p.initiator_wwn,
	}
	//this will store returned result
	var result result
	//call remote procedure with args
	err = client.Call("export_destroy", args, &result)
	return err
}

func (p *iscsiProvisioner) vol_create(name string, size int64, pool string) error {

	client, err := p.getConnection()
	defer client.Close()
	if err != nil {
		log.Warnln(err)
		return err
	}

	//make arguments object
	args := vol_createArgs{
		pool: pool,
		name: name,
		size: size,
	}
	//this will store returned result
	var result result
	//call remote procedure with args
	err = client.Call("vol_create", args, &result)
	return err
}

func (p *iscsiProvisioner) export_create(vol string, lun int32, pool string) error {

	client, err := p.getConnection()
	defer client.Close()
	if err != nil {
		log.Warnln(err)
		return err
	}

	//make arguments object
	args := export_createArgs{
		pool:          pool,
		vol:           vol,
		initiator_wwn: p.initiator_wwn,
		lun:           lun,
	}
	//this will store returned result
	var result result
	//call remote procedure with args
	err = client.Call("export_create", args, &result)
	return err
}

func (p *iscsiProvisioner) export_list() (exportList, error) {

	client, err := p.getConnection()
	defer client.Close()
	if err != nil {
		log.Warnln(err)
		return nil, err
	}

	//this will store returned result
	var result exportList
	//call remote procedure with args
	err = client.Call("export_list", nil, &result)
	return result, err
}

func (slice exportList) Len() int {
	return len(slice)
}

func (slice exportList) Less(i, j int) bool {
	return slice[i].lun < slice[j].lun
}

func (slice exportList) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func (p *iscsiProvisioner) getConnection() (*jsonrpc2.Client, error) {
	log.Debugln("opening connection to targetd: ", p.targetdURL)

	client := jsonrpc2.NewHTTPClient(p.targetdURL)

	if client == nil {
		log.Warnln("error creating the connection to targetd", p.targetdURL)
		return nil, errors.New("error creating the connection to targetd")
	}
	log.Debugln("targetd client created")
	return client, nil
}

//func (p *iscsiProvisioner) getConnection2() (*rpc.Client, error) {
//	log.Debugln("opening connection to targetd: ", p.targetdURL)
//
//	client, err := jsonrpc.Dial("tcp", p.targetdURL)
//
//	if err != nil {
//		log.Warnln(err)
//		return nil, err
//	}
//	log.Debugln("targetd client created")
//	return client, nil
//}
