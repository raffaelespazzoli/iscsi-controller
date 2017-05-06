# iscsi provisioner 
iscsi provisioner is a out of tree provisioner for iscsi storage for Kubernetes and OpenShift.

## Prerequisites

iscsi provisioner has the following prerequisistes:

1. an iscsi server managed by `targetcli`
2. all the openshift nodes correclty configured to communicate with the iscsi server
3. targetd installed on the iscsi server and correclty configured
4. sufficient disk space available as volume group (vg are the only supported backing storage at the momment)

## how it works

when a pvc request is issued for an iscsi provisioner controlled storage class the following happens:

1. a new volume in the configured volume group is created, the size of the volume corresponds to the size requested in the pvc
2. the volume is exported to the first available lun and made accessible to all the configured initiators.
3. the corresponding pv is created and bound to the pvc. 


Each storage class is tied to an iscsi iqn and a volume group. Because an iqn can manage a maximum of 255 luns, each storage class manage at most 255 pvs. iscsi provisioner can manage multiple storage classes.

## installing the prerequisites

### configure the iscsi server

#### install target

```
sudo yum install -y targetcli targetd

```

#### configure target

```
sudo systemctl enable target
sudo systemctl start target
```

#### create a volume group
the volume group should be called `vg-target`, this way you don' have to change any default

here is how you would do it in minishift
```
cd /var/lib/minishift
sudo dd if=/dev/zero of=disk.img bs=1G count=2
export LOOP=`sudo losetup -f`
sudo losetup $LOOP disk.img
sudo vgcreate vg-targetd $LOOP
```

#### configure targetd

choose a password for `/etc/target/targetd.yaml`

```
sudo systemctl enable targetd
sudo systemctl start targetd
```


### configure the nodes (iscsi clients)

do the following for each node

#### install the required packages
These should be available in a standard openshift installation
```
sudo yum install -y iscsi-initiator-utils
```
#### configure the initiator name

edit this file `/etc/iscsi/initiatorname.iscsi` and add an initiator name in each

### install the iscsi provisioner pod
run the following commands. The secret correspond to username and password you have chosen for targetd (admin is the default for the username)
```
oc new-project iscsi-provisioner
oc create sa iscsi-provisioner
oc adm policy add-cluster-role-to-user cluster-reader system:serviceaccount:iscsi-provisioner:iscsi-provisioner
oc adm policy add-cluster-role-to-user system:pv-provisioner-controller system:serviceaccount:iscsi-provisioner:iscsi-provisioner
oc adm policy add-cluster-role-to-user system:pv-binder-controller system:serviceaccount:iscsi-provisioner:iscsi-provisioner
oc adm policy add-cluster-role-to-user system:pv-recycler-controller system:serviceaccount:iscsi-provisioner:iscsi-provisioner
oc secret new-basicauth targetd_account --username=admin --password=ciao
oc create -f https://raw.githubusercontent.com/raffaelespazzoli/iscsi-controller/master/openshift/iscsi-provisioner-dc.yaml
```
### create a storage class
storage classes should look like the following
```
kind: StorageClass
apiVersion: storage.k8s.io/v1beta1
metadata:
  name: iscsi
provisioner: iscsi
parameters:
# this id where the iscsi server is running
  targetPortal: 192.168.99.100:3260
  
# this is the iscsi server iqn  
  iqn: iqn.2003-01.org.linux-iscsi.minishift:targetd
  
# this is the iscsi interface to be used, the default is default
# iscsiInterface: default

# this must be on eof the volume groups condifgured in targed.yaml, the default is vg-targetd
# volumeGroup: vg-targetd

# this is a comma separated list of initiators that will be give access to the created volumes, they must correspond to what you have configured in your nodes.
  initiators: iqn.2014-06.com.example:desktop0 
```
you can create one with the following command

```
oc create -f https://raw.githubusercontent.com/raffaelespazzoli/iscsi-controller/master/openshift/iscsi-provisioner-dc.yaml
```
### test iscsi provisioner
create a pvc
```
oc create -f https://raw.githubusercontent.com/raffaelespazzoli/iscsi-controller/master/openshift/iscsi-provisioner-pvc.yaml
```
verify that the pv has been created
```
oc get pv
```
you may also want to verify that the volume has been created in you volume group
```
targetcli ls
```
deploy a pod that uses the pvc
```
oc create -f https://raw.githubusercontent.com/raffaelespazzoli/iscsi-controller/master/openshift/iscsi-test-pod.yaml
```
