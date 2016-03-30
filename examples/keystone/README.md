Gohan & Keystone integraion example
------------------------------------

We show how to integrate Gohan & Keystone in this example


Install Keystone
-----------------

1. Install python & pip
2. Setup keystone with CORS support

This is sample shell code for dev setup

``` shell
GOHAN_APP_DIR=`pwd`

git clone https://github.com/openstack/keystone.git
cd keystone/
pip install wsgicors
pip install tox
tox -epy

cp $GOHAN_APP_DIR/keystone-paste.ini etc
cp etc/keystone.conf{.sample,}

.tox/py/bin/keystone-manage db_sync
.tox/py/bin/keystone-all

pip install python-opensatckclient
export OS_TOKEN=ADMIN
tools/sample_data.sh

openstack user create gohan --project service\
                      --password "gohan"

openstack role add --user gohan \
                   --project service \
                   admin
```

Test keystone

``` shell
export OS_AUTH_URL=http://localhost:5000/v2.0
export OS_PROJECT_NAME=demo
export OS_USERNAME=admin
export OS_PASSWORD=secrete

openstack project list
```

Configure gohan
----------------

You need set auth_url and keystone/fake=false

``` yaml
keystone:
    use_keystone: true
    fake: false
    auth_url: "http://localhost:5000/v2.0"
    user_name: "gohan"
    tenant_name: "service"
    password: "gohan"
```

Configure webui config.json
-----------------------------

You need set auto_url in config.json on webui

``` json
{
    "auth_url": "http://localhost:5000/v2.0",
    "gohan": {
        "schema": "/gohan/v0.1/schemas",
        "url": "http://__HOST__"
    }
}
```

Note that we can't mix http & https in webui & keystone api endpoint.
