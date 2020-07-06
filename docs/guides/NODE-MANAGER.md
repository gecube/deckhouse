---
title: "Управление узлами кластера"
tags:
  - node-manager
type:
  - instruction
search: node-manager, добавить ноду в кластер
permalink: /guides/node-manager.html
---

## Введение

Модуль `node-manager` управляет узлами кластера. Более подробно о модуле можно узнать в документации [модуля node-manager]({{ site.baseurl }}/modules/040-node-manager/).

### Как автоматически добавить статичный узел в кластер?

Чтобы добавить новую ноду в статичный кластер необходимо:
- Создать `NodeGroup` с необходимыми параметрами (`nodeType` может быть `Static` или `Hybrid`) или использовать уже существующую. К примеру создадим `NodeGroup` с именем `example`.
- Получить скрипт для установки и настройки ноды: `kubectl -n d8-cloud-instance-manager get secret manual-bootstrap-for-example -o json | jq '.data."bootstrap.sh"' -r`
- Перед настройкой kubernetes на ноде убедитесь, что вы выполнили все необходимые действия для корректной работы узла в кластере:
  - Добавили в `/etc/fstab` все необходимые маунты (nfs, ceph, ...)
  - Установили на ноду `ceph-common` нужной версии или еще какие-то пакеты
  - Настроили сеть для коммуникации узлов в кластере
- Зайти на новую ноду по ssh и выполнить команду из секрета: `echo <base64> | base64 -d | bash`

### Как завести ноду под управление node-manager?

Чтобы завести ноду под управление `node-manager`:
- Создать `NodeGroup` с необходимыми параметрами (`nodeType` может быть `Static` или `Hybrid`) или использовать уже существующую. К примеру создадим `NodeGroup` с именем `nodes`.
- Получить скрипт для установки и настройки моды: `kubectl -n d8-cloud-instance-manager  get secret manual-bootstrap-for-nodes-o json | jq '.data."adopt.sh"' -r`
- Зайти на новую ноду по ssh и выполнить команду из секрета: `echo <base64> | base64 -d | bash`

### Как вывести ноду из под управления node-manager?

- Остановить сервис и таймер bashible: `systemctl stop bashible.timer bashible.service`
- Удалить скрипты bashible: `rm -rf /var/lib/bashible`
- Удалить с ноды аннотации и лейблы: 
```shell script
kubectl annotate node <node_name> node.deckhouse.io/configuration-checksum- update.node.deckhouse.io/waiting-for-approval- update.node.deckhouse.io/disruption-approved- update.node.deckhouse.io/disruption-required- update.node.deckhouse.io/approved- update.node.deckhouse.io/draining- update.node.deckhouse.io/drained-
kubectl label node <node_name> node.deckhouse.io/group-
```

### Как понять, что что-то пошло не так?

Модуль `node-manager` создает на каждой ноде сервис `bashible` и его логи можно посмотреть: `journalctl -fu bashible`.

### Как посмотреть, что в данный момент выполняется на ноде при ее создании?

Если мы хотим узнать, что происходит на ноде (к примеру она долго создается), то можно посмотреть логи `cloud-init` для этого необходимо:
- Найти ноду, которая сейчас бутстрапится: `kubectl -n d8-cloud-instance-manager  get machine | grep Pending`
- Посмотреть информацию о `machine`: `kubectl -n d8-cloud-instance-manager describe machine kube-2-worker-01f438cf-757f758c4b-r2nx2`
В дескрайбе мы увидим такую информацию:
```shell script
Status:
  Bootstrap Status:
    Description:   Use 'nc 192.168.199.115 8000' to get bootstrap logs.
    Tcp Endpoint:  192.168.199.115
```
- Выполнить команду `nc 192.168.199.115 8000`, тогда вы увидите логи `cloud-init` и увидите на чем зависла настройка ноды

Логи первоначальной настройки ноды находятся в `/var/log/cloud-init-output.log`.

### Как настроить ноду с GPU?

Если у вас есть нода с GPU и вы хотите настроить docker для работы с `node-manager`, то вам необходимо выполнить все настройки на ноде по [документации](https://github.com/NVIDIA/k8s-device-plugin#quick-start).

Создать `NodeGroup` с такими параметрами:
```shell script
  docker:
    manage: false
  operatingSystem:
    manageKernel: false
```

После чего добавить ноду под управление `node-manager`.

### Какие параметры NodeGroup к чему приводят?

| Параметр NG                   | Disruption update    | Перезаказ нод | Рестарт kubelet |
|-------------------------------|----------------------|---------------|-----------------|
| operatingSystem.manageKernel  | + (true) / - (false) | -             | -               |
| kubelet.maxPods               | -                    | -             | +               |
| kubelet.rootDir               | -                    | -             | +               |
| docker.maxConcurrentDownloads | +                    | -             | +               |
| docker.manage                 | + (true) / - (false) | -             | -               |
| nodeTemplate                  | -                    | -             | -               |
| chaos                         | -                    | -             | -               |
| kubernetesVersion             | -                    | -             | +               |
| static                        | -                    | -             | +               |
| disruptions                   | -                    | -             | -               |
| cloudInstances.classReference | -                    | +             | -               |

Подробно о всех параметрах можно прочитать в [документации модуля](https://github.com/deckhouse/deckhouse/tree/master/modules/040-node-manager/docs#nodegroup-custom-resource)

В случае изменения параметра `instancePrefix` в конфигурации deckhouse не будет происходить `RollingUpdate`. Deckhouse создаст новые `MachineDeployment`, а старые удалит.