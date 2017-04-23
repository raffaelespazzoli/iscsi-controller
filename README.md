# install target

```
sudo yum install -y targetcli targetd

```

# configure target

```
sudo systemctl enable target
sudo systemctl start target
```

# configure targetd

choose a password for `/etc/target/targetd.yaml`

```
sudo dd if=/dev/zero of=disk.img bs=1G count=2
export LOOP=`losetup -f`
sudo losetup $LOOP disk.img
sudo vgcreate vg-targetd $LOOP
sudo systemctl enable targetd
sudo systemctl start targetd
```


# configure targetd