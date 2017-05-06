FROM registry.access.redhat.com/rhel7:latest
LABEL authors="Raffaele Spazzoli <rspazzol@redhat.com>" 

#RUN INSTALL_PKGS="targetcli targetd python3-lvm rsyslog" && \
#    yum -y update && \
#    yum install -y  $INSTALL_PKGS && \
#    yum clean all && \
#    systemctl enable target && \
#    systemctl enable targetd
    

# RUN cp /usr/lib/systemd/system/dbus.service /etc/systemd/system/; sed -i 's/OOMScoreAdjust=-900//' /etc/systemd/system/dbus.service

#ENV container=docker

EXPOSE 3260 18700

VOLUME ["/run", "/tmp", "/iscsi-volume"]

#CMD [ "/sbin/init", "--log-target=console", "--default-standard-output=syslog+console", "--default-standard-error=syslog+console" ]    
CMD [ "/sbin/init"]    