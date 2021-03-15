{{- if eq .cri "Docker" }}

bb-event-on 'bb-package-installed' 'post-install'
post-install() {
  if bb-flag? there-was-containerd-installed; then
    bb-log-info "Setting reboot flag due to containerd package was updated"
    bb-flag-set reboot
    bb-flag-unset there-was-containerd-installed
  fi

  if bb-flag? there-was-docker-installed; then
    bb-log-info "Setting reboot flag due to docker package was updated"
    bb-flag-set reboot
    bb-flag-unset there-was-docker-installed
  fi

  if bb-is-ubuntu-version? 18.04; then
    systemctl unmask docker.service  # Fix bug in ubuntu 18.04: https://bugs.launchpad.net/ubuntu/+source/docker.io/+bug/1844894
  fi
  systemctl enable docker.service
{{ if ne .runType "ImageBuilding" -}}
  systemctl restart docker.service
{{- end }}
}

# TODO: remove ASAP, provide proper migration from "docker.io" to "docker-ce"
if bb-apt-package? docker.io ; then
  bb-log-warning 'Skipping "docker-ce" installation, since "docker.io" is already installed'
  exit 0
fi

if bb-apt-package? containerd.io && ! bb-apt-package? docker-ce ; then
  bb-deckhouse-get-disruptive-update-approval
  systemctl stop kubelet.service
  systemctl stop containerd.service
  # Kill running containerd-shim processes
  kill $(ps ax | grep containerd-shim | grep -v grep |awk '{print $1}') 2>/dev/null || true
  # Remove mounts
  umount $(mount | grep "/run/containerd" | cut -f3 -d" ") 2>/dev/null || true
  bb-apt-remove containerd.io
  rm -rf /var/lib/containerd/ /var/run/containerd /usr/local/bin/crictl
  # Pod kubelet-eviction-thresholds-exporter in cri=Containerd mode mounts /var/run/docker.sock, /var/run/docker.sock will be a directory and newly installed docker won't run.
  rm -rf /var/run/docker.sock
  bb-log-info "Setting reboot flag due to cri being updated"
  bb-flag-set reboot
fi

if bb-is-ubuntu-version? 20.04 ; then
  desired_version_containerd="containerd.io=1.4.3-1"
  allowed_versions_containerd_pattern="containerd.io=1.[23]"
  desired_version_docker="docker-ce=5:19.03.13~3-0~ubuntu-focal"
  allowed_versions_docker_pattern=""
elif bb-is-ubuntu-version? 18.04 ; then
  desired_version_containerd="containerd.io=1.4.3-1"
  allowed_versions_containerd_pattern="containerd.io=1.[23]"
{{- if eq .kubernetesVersion "1.19" }}
  desired_version_docker="docker-ce=5:19.03.13~3-0~ubuntu-bionic"
  allowed_versions_docker_pattern="docker-ce=5:18.09.7~3-0~ubuntu-bionic"
{{- else }}
  desired_version_docker="docker-ce=5:18.09.7~3-0~ubuntu-bionic"
  allowed_versions_docker_pattern=""
{{- end }}
elif bb-is-ubuntu-version? 16.04 ; then
  desired_version_containerd="containerd.io=1.4.3-1"
  allowed_versions_containerd_pattern="containerd.io=1.[23]"
  desired_version_docker="docker-ce=5:18.09.7~3-0~ubuntu-xenial"
  allowed_versions_docker_pattern=""
else
  bb-log-error "Unsupported Ubuntu version"
  exit 1
fi

should_install_containerd=true
version_in_use="$(dpkg -l containerd.io 2>/dev/null | grep -E "(hi|ii)\s+(containerd.io)" | awk '{print $2"="$3}' || true)"
if test -n "$allowed_versions_containerd_pattern" && test -n "$version_in_use" && grep -Eq "$allowed_versions_containerd_pattern" <<< "$version_in_use"; then
  should_install_containerd=false
fi

if [[ "$version_in_use" == "$desired_version_containerd" ]]; then
  should_install_containerd=false
fi

if [[ "$should_install_containerd" == true ]]; then

  if bb-apt-package? "$(echo $desired_version_containerd | cut -f1 -d"=")"; then
    bb-flag-set there-was-containerd-installed
  fi

  bb-deckhouse-get-disruptive-update-approval
  bb-apt-install $desired_version_containerd
fi

should_install_docker=true
version_in_use="$(dpkg -l docker-ce 2>/dev/null | grep -E "(hi|ii)\s+(docker-ce)" | awk '{print $2"="$3}' || true)"
if test -n "$allowed_versions_docker_pattern" && test -n "$version_in_use" && grep -Eq "$allowed_versions_docker_pattern" <<< "$version_in_use"; then
  should_install_docker=false
fi

if [[ "$version_in_use" == "$desired_version_docker" ]]; then
  should_install_docker=false
fi

if [[ "$should_install_docker" == true ]]; then
  desired_version_docker_cli="$(sed 's/docker-ce/docker-ce-cli/' <<< "$desired_version_docker")"

  if bb-apt-package? "$(echo $desired_version_docker | cut -f1 -d"=")"; then
    bb-flag-set there-was-docker-installed
  fi

  bb-deckhouse-get-disruptive-update-approval

  if bb-apt-package? docker.io; then
    bb-apt-remove docker.io
    bb-flag-set there-was-docker-installed
  fi

  bb-apt-install $desired_version_docker $desired_version_docker_cli
fi

{{- end }}
