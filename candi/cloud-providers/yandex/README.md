---
title: Cloud provider - Yandex
sidebar: candi
hide_sidebar: false
---

Схема размещения описывается объектом `YandexClusterConfiguration`. Его поля:

* `layout` — архитектура расположения ресурсов в облаке.
  * Варианты — `Standard`, `WithoutNAT` или `WithNATInstance` (описание ниже).
* `withNATInstance` — настройки для layout'а `WithNATInstance`.
  * `natInstanceExternalAddress` — внешний [зарезервированный белый IP адрес](#Резервирование-белого-IP-адреса) или адрес из `externalSubnetID` при указании опции.
  * `internalSubnetID` — ID подсети для внутреннего интерфейса
  * `externalSubnetID` — при указании данной опции к узлу будет подключен дополнительный сетевой интерфейс, в который будет идти маршрут по-умолчанию.
* `provider` — параметры подключения к API Yandex.Cloud.
  * `cloudID` — идентификатор облака.
  * `folderID` — идентификатор директории.
  * `serviceAccountJSON` — JSON, выдаваемый [yc iam key create](#права)
* `masterNodeGroup` — спеки для описания NG мастера.
  * `replicas` — сколько мастер-узлов создать.
  * `instanceClass` — частичное содержимое полей [YandexInstanceClass](/modules/030-cloud-provider-yandex/docs/README.md#yandexinstanceclass-custom-resource). Обязательными параметрами являются `cores`, `memory`, `imageID`. Допустимые параметры:
    * `cores`
    * `memory`
    * `imageID`
    * `externalIPAddresses` — список внешних адресов. Количество элементов массива должно соответствовать `replicas`.
      * При отсутствии опции `externalSubnetID` нужно использовать или [зарезервированные белые IP адреса](#Резервирование-белого-IP-адреса) или константу `Auto`.
      * При наличии опции `externalSubnetID` необходимо выбрать конкретные свободные IP из указанной подсети.
    * `externalSubnetID` — при указании данной опции к узлу будет подключен дополнительный сетевой интерфейс, в который будет идти маршрут по-умолчанию.
    * `zones` — узлы будут создаваться только в перечисленных зонах.
* `nodeGroups` — массив дополнительных NG для создания статичных узлов (например, для выделенных фронтов или шлюзов). Настройки NG:
  * `name` — имя NG, будет использоваться для генерации имени нод.
  * `replicas` — сколько узлов создать.
  * `instanceClass` — частичное содержимое полей [YandexInstanceClass](/modules/030-cloud-provider-yandex/docs/README.md#yandexinstanceclass-custom-resource). Обязательными параметрами являются `cores`, `memory`, `imageID`. Допустимые параметры:
    * `cores`
    * `memory`
    * `imageID`
    * `coreFraction`
    * `externalIPAddresses` — список внешних адресов. Количество элементов массива должно соответствовать `replicas`.
      * При отсутствии опции `externalSubnetID` нужно использовать или [зарезервированные белые IP адреса](#Резервирование-белого-IP-адреса) или константу `Auto`.
      * При наличии опции `externalSubnetID` необходимо выбрать конкретные свободные IP из указанной подсети.
    * `externalSubnetID` — при указании данной опции к узлу будет подключен дополнительный сетевой интерфейс, в который будет идти маршрут по-умолчанию.
    * `zones` — узлы будут создаваться только в перечисленных зонах.
* `nodeNetworkCIDR` — данная подсеть будет разделена на **три** равных части и использована для создания подсетей в трёх зонах Yandex.Cloud.
* `existingNetworkID` — существующей VPC Network.
* `dhcpOptions` — список DHCP опций, которые будут установлены на все подсети.
  * `domainName` — search домен.
  * `domainNameServers` — список адресов рекурсивных DNS.
* `sshKey` — публичный ключ для доступа на ноды.

### Standard

В данной схеме размещения узлы не будут иметь публичных адресов, а будут выходить в интернет через Yandex.Cloud NAT.

![resources](https://docs.google.com/drawings/d/e/2PACX-1vTSpvzjcEBpD1qad9u_UgvsOrYT_Xtnxwg6Pzb64HQHLqQWcZi6hhCNRPKVUdYKX32nXEVJeCzACVRG/pub?w=812&h=655)
<!--- Исходник: https://docs.google.com/drawings/d/1WI8tu-QZYcz3DvYBNlZG4s5OKQ9JKyna7ESHjnjuCVQ/edit --->

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: YandexClusterConfiguration
layout: Standard
provider:
  cloudID: dsafsafewf
  folderID: enh1233214367
  serviceAccountJSON: |
    {"test": "test"}
masterNodeGroup:
  replicas: 1
  instanceClass:
    cores: 4
    memory: 8192
    imageID: testtest
    externalIPAddresses:
    - "198.51.100.5"
    - "Auto"
    externalSubnetID: tewt243tewsdf
    zones:
    - ru-central1-a
    - ru-central1-b
nodeGroups:
- name: khm
  replicas: 1
  instanceClass:
    cores: 4
    memory: 8192
    imageID: testtest
    coreFraction: 50
    externalIPAddresses:
    - "198.51.100.5"
    - "Auto"
    externalSubnetID: tewt243tewsdf
    zones:
    - ru-central1-a
sshKey: "ssh-rsa ewasfef3wqefwefqf43qgqwfsd"
nodeNetworkCIDR: 192.168.12.13/24
existingNetworkID: tewt243tewsdf
dhcpOptions:
  domainName: test.local
  domainNameServers:
  - 213.177.96.1
  - 231.177.97.1
```

#### Включение Cloud NAT

**Внимание!** Сразу же (в течение 3х минут) после создания базовых сетевых ресурсов, для всех подсетей необходимо включить Cloud NAT. Вручную через web-интерфейс. Если этого не сделать, то bootstrap процесс не сможет завершится. Для того, чтобы появилась опция включения Cloud NAT, необходимо заранее (за неделю) обратится в саппорт Yandex.Cloud.

![Включение NAT](docs/img/enable_cloud_nat.png)

### WithoutNAT

В данной схеме размещения NAT (любого вида) не используется, а каждому узлу выдаётся публичный IP.

**Внимание!** В модуле cloud-provider-yandex пока нет поддержки Security Groups, поэтому все ноды кластера будут смотреть наружу.

![resources](https://docs.google.com/drawings/d/e/2PACX-1vTgwXWsNX6CKCRaMf5t6rl3kpKQQFHK6T8Dsg1jAwAwYaN1MRbxKFsSFQHeo1N3Qec4etPpeA0guB6-/pub?w=812&h=655)
<!--- Исходник: https://docs.google.com/drawings/d/1I7M9DquzLNu-aTjqLx1_6ZexPckL__-501Mt393W1fw/edit --->

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: YandexClusterConfiguration
layout: WithoutNAT
provider:
  cloudID: dsafsafewf
  folderID: enh1233214367
  serviceAccountJSON: |
    {"test": "test"}
masterNodeGroup:
  replicas: 1
  instanceClass:
    cores: 4
    memory: 8192
    imageID: testtest
    externalIPAddresses:
    - "198.51.100.5"
    - "Auto"
    externalSubnetID: tewt243tewsdf
    zones:
    - ru-central1-a
    - ru-central1-b
nodeGroups:
- name: khm
  replicas: 1
  instanceClass:
    cores: 4
    memory: 8192
    imageID: testtest
    coreFraction: 50
    externalIPAddresses:
    - "198.51.100.5"
    - "Auto"
    externalSubnetID: tewt243tewsdf
    zones:
    - ru-central1-a
sshKey: "ssh-rsa ewasfef3wqefwefqf43qgqwfsd"
nodeNetworkCIDR: 192.168.12.13/24
existingNetworkID: tewt243tewsdf
dhcpOptions:
  domainName: test.local
  domainNameServers:
  - 213.177.96.1
  - 231.177.97.1
```

### WithNATInstance

В данной схеме размещения создаётся NAT instance, а в таблицу маршрутизации добавляется правило на 0.0.0.0/0 с NAT instance nexthop'ом.

![resources](https://docs.google.com/drawings/d/e/2PACX-1vSnNqebgRdwGP8lhKMJfrn5c0QXDpe9YdmIlK4eDberysLLgYiKNuwaPLHcyQhJigvQ21SANH89uipE/pub?w=812&h=655)
<!--- Исходник: https://docs.google.com/drawings/d/1oVpZ_ldcuNxPnGCkx0dRtcAdL7BSEEvmsvbG8Aif1pE/edit --->

```yaml
apiVersion: deckhouse.io/v1alpha1
kind: YandexClusterConfiguration
layout: WithNATInstance
withNATInstance:
  natInstanceExternalAddress: 30.11.34.45
  internalSubnetID: sjfwefasjdfadsfj
  externalSubnetID: etasjflsjdfiorej
provider:
  cloudID: dsafsafewf
  folderID: enh1233214367
  serviceAccountJSON: |
    {"test": "test"}
masterNodeGroup:
  replicas: 1
  instanceClass:
    cores: 4
    memory: 8192
    imageID: testtest
    externalIPAddresses:
    - "198.51.100.5"
    - "Auto"
    externalSubnetID: tewt243tewsdf
    zones:
    - ru-central1-a
    - ru-central1-b
nodeGroups:
- name: khm
  replicas: 1
  instanceClass:
    cores: 4
    memory: 8192
    imageID: testtest
    coreFraction: 50
    externalIPAddresses:
    - "198.51.100.5"
    - "Auto"
    externalSubnetID: tewt243tewsdf
    zones:
    - ru-central1-a
sshKey: "ssh-rsa ewasfef3wqefwefqf43qgqwfsd"
nodeNetworkCIDR: 192.168.12.13/24
existingNetworkID: tewt243tewsdf
dhcpOptions:
  domainName: test.local
  domainNameServers:
  - 213.177.96.1
  - 231.177.97.1
```

## Права

Service account key неудобно создавать через Terraform или web-интерфейс, потому что только `yc` выдаёт корректно отформатированный JSON с ключом.

```shell
$ yc iam service-account create --name candi
id: ajee8jv6lj8t7eg381id
folder_id: b1g1oe1s72nr8b95qkgn
created_at: "2020-08-17T08:50:38Z"
name: candi

$ yc resource-manager folder add-access-binding prod --role editor --subject serviceAccount:ajee8jv6lj8t7eg381id

$ yc iam key create --service-account-name candi --output candi-sa-key.json
```

## Резервирование белого IP-адреса

Для использования в `externalIPAddresses` и `natInstanceExternalAddress`.

```shell
$ yc vpc address create --external-ipv4 zone=ru-central1-a
id: e9b4cfmmnc1mhgij75n7
folder_id: b1gog0h9k05lhqe5d88l
created_at: "2020-09-01T09:29:33Z"
external_ipv4_address:
  address: 178.154.226.159
  zone_id: ru-central1-a
  requirements: {}
reserved: true
```