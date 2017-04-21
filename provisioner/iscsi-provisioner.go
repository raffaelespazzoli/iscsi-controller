package provisioner

import (
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"k8s.io/client-go/pkg/api/v1"
	"net/rpc"
	"net/rpc/jsonrpc"
)

var log = logrus.New()

type vol_createArgs struct {
  pool string
  name string
  size int
}

type vol_destroyArgs struct {
  pool string
  name string
}

type export_createArgs struct {
  pool string,
  vol string
  initiator_wwn string 
  lun string
}

type export_destroyArgs struct {
  pool string
  vol string
  initiator_wwn string
}

type iscsiProvisioner struct {
	client rpc.Client
	pool string
	initiator_wwn string
}

func NewiscsiProvisioner(url string, pool string) controller.Provisioner {

  initLog()
	client, err := jsonrpc.Dial(network, url)

	if err != nil {
		log.Fatalln(err)
	}
	log.Debugln("targetd client created")

	return &iscsiProvisioner{
		pvDir: client,
		pool:  pool,
	}
}

// Provision creates a storage asset and returns a PV object representing it.
func (p *iscsiProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	lun, vol := p.createVolume(options)
	if err != nil {
		return nil, err
	}

	annotations := make(map[string]string)
	annotations["volume_name"] = vol
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
			  ISCSI: &v1.ISCSIVolumeSource {
			    TargetPortal: options.Parameters["targetPortal"],
			    IQN: options.Parameters["iqn"],
			    ISCSIInterface: options.Parameters["iscsiInterface"], 
			    Lun: options.PVName,
			    ReadOnly: false,
			    FSType: "zfs",
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
  export_destroy(p.pool, vol, p.initiator_wwn)
  vol_destroy(p.pool, options.PVName)

	return pv, nil
}

func initLog() {
	var err error
	log.Level, err = logrus.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		log.Fatalln(err)
	}
}

func (p *iscsiProvisioner) createVolume(options controller.VolumeOptions) (lun string, vol string) {  
  vol:=vol_create(p.pool, options.PVName, size)
  export_create(p.pool, vol, p.initiator_wwn, options.PVName)
} 
