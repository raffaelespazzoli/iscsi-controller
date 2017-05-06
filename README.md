# iscsi provisioner 

iscsi provisioner is a out of tree provisioner for iSCSI storage for
Kubernetes and OpenShift.  The provisioniner uses the API provided by
[targetd](https://github.com/open-iscsi/targetd) to create and export
iSCSI storage on a remote server.

## Prerequisites

iscsi provisioner has the following prerequisistes:

1. an iSCSI server managed by `targetd`
2. all the openshift nodes correclty configured to communicate with the iSCSI server
3. sufficient disk space available as LVM2 volume group (vg are the only supported backing storage at the momment)

## How it works

when a pvc request is issued for an iscsi provisioner controlled
storage class the following happens:

1. a new volume in the configured volume group is created, the size of
the volume corresponds to the size requested in the pvc
2. the volume is exported to the first available lun and made
accessible to all the configured initiators.
3. the corresponding pv is created and bound to the pvc.


Each storage class is tied to an iSCSI target and a volume
group. Because an target can manage a maximum of 255 LUNs, each
storage class manage at most 255 pvs. iscsi provisioner can manage
multiple storage classes.

## Installing the prerequisites

These instructions should work for RHEL/CentOS 7+ and Fedora 24+.

### A note about names

In various places, iSCSI Qualified Names (IQNs) need to be created.
These need to be unique.  So every target must have it's own unique
IQN, and every client (initiator) must have its own IQN.


IF NON-UNIQUE IQNs ARE USED, THEN THERE IS A POTENTIAL FOR DATA LOSS
AND BAD PERFORMANCE!

IQNs have a specific format:

iqn.YEAR-MM.com.example.blah:tag

See the [wikipedia
article](https://en.wikipedia.org/wiki/ISCSI#Addressing) for more
information.

### Configure Storage

Before configuring the iSCSI server, it needs to have storage
configured.  `targetd` uses LVM to provision storage.

If possible, it's best to have a dedicated disk or partition that can
be configured as a volume group.  However, if this is not possible, a
loopback device can be used to simulate a dedicated block device.

#### Create a Volume Group with a dedicated disk or partition

This requires an additional dedicated disk or partition to use for the
volume group.  If that's not possible, see the section on using a
loopback device.

Assuming that the dedicated block device is `/dev/vdb` and that
`targetd` is configured to use `vg-targetd`:

```
pvcreate /dev/vdb
vgcreate vg-targetd /dev/vdb
```

#### Create a Volume Group on a Loopback Device
the volume group should be called `vg-target`, this way you don' have to change any default

here is how you would do it in minishift
```
cd /var/lib/minishift
sudo dd if=/dev/zero of=disk.img bs=1G count=2
export LOOP=`sudo losetup -f`
sudo losetup $LOOP disk.img
sudo vgcreate vg-targetd $LOOP
```

### Configure the iSCSI server

#### Install targetd and targetcli

Only `targetd` needs to be installed.  However, it's highly recommended
to also install `targetcli` as it provides a simple user interface for
looking at the state of the iSCSI system.

```
sudo yum install -y targetcli targetd rsyslog

```

#### Configure target

Enable and start `target.service`.  This will ensure that iSCSI
configuration persists through reboot.

```
sudo systemctl enable target
sudo systemctl start target
```

#### Configure targetd

First, edit `/etc/target/targetd.yaml`.  A working sample
configuration is provided below:

```
password: ciao

# defaults below; uncomment and edit
pool_name: vg-targetd
user: admin
ssl: false
target_name: iqn.2003-01.org.linux-iscsi.minishift:targetd
```

Next, enable and start `targetd.service`.

```
sudo systemctl enable targetd
sudo systemctl start targetd
```

#### Configure the Firewall

The default configuration requires that port 3260/tcp, 3260/udp and
18700/tcp be open on the iSCSI server.

If using `firewalld`, 

```
firewall-cmd --add-service=iscsi-target --permanent
firewall-cmd --add-port=18700/tcp --permanent 
firewall-cmd --reload
```

Otherwise, add the following iptables rules to `/etc/sysconfig/iptables`

```
TODO
```

### Configure the nodes (iscsi clients)

#### Install the iscsi-initiator-utils package

The `iscsiadm` command is required for all clients.  This is provided
by the `iscsi-initiator-utils` package and should be part of the
standard RHEL, CentOS or Fedora installation.

```
sudo yum install -y iscsi-initiator-utils
```

#### Configure the Initiator Name

Each node requires a unique initiator name.  USE OF DUPLICATE NAMES
MAY CAUSE PERFORMANCE ISSUES AND DATA LOSS.

By default, a random initiator name is generated when the
`iscsi-initiator-utils` package is installed.  This usually unique
enough, but is not guaranteed.  It's also not very descriptive.

To set a custom initiator name, edit the file `/etc/iscsi/initiatorname.iscsi`:

```
InitiatorName=iqn.2017-04.com.example:node1
```

In the above example, the initiator name is set to
`iqn.2017-04.com.example:node1`.

After changing the initiator name, restart `iscsid.service`.

```
sudo systemctl restart iscsid
```

### Install the iscsi provisioner pod

run the following commands. The secret correspond to username and password you have chosen for targetd (admin is the default for the username)
```
oc new-project iscsi-provisioner
oc create sa iscsi-provisioner
oc adm policy add-cluster-role-to-user cluster-reader system:serviceaccount:iscsi-provisioner:iscsi-provisioner
oc adm policy add-cluster-role-to-user system:pv-provisioner-controller system:serviceaccount:iscsi-provisioner:iscsi-provisioner
oc adm policy add-cluster-role-to-user system:pv-binder-controller system:serviceaccount:iscsi-provisioner:iscsi-provisioner
oc adm policy add-cluster-role-to-user system:pv-recycler-controller system:serviceaccount:iscsi-provisioner:iscsi-provisioner
oc secret new-basicauth targetd-account --username=admin --password=ciao
oc create -f https://raw.githubusercontent.com/raffaelespazzoli/iscsi-controller/master/openshift/iscsi-provisioner-dc.yaml
```
### Create a storage class
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
  initiators: iqn.2017-04.com.example:node1 
```
you can create one with the following command

```
oc create -f https://raw.githubusercontent.com/raffaelespazzoli/iscsi-controller/master/openshift/iscsi-provisioner-class.yaml
```
### Test iscsi provisioner
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
