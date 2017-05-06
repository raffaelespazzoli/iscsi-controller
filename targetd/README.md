# install targetd in a container

## build the image

```
oc new-project targetd
oc new-build --strategy=docker --binary=true --name=targetd 
oc start-build targetd --from-file . -F
```

## run the image

```
docker run  -ti --privileged -p 13260:3260 -p 18700 --cap-add=ALL -v /usr/lib/modules:/usr/lib/modules -v /sys/kernel/config:/sys/kernel/config raffaelespazzoli/fedora-systemd:latest
```

```
oc adm policy add-scc-to-user privileged -z default
```