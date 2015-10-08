==============
Installation
==============

How to install gohan
------------------------------------


+ Install make
+ Setup go env http://golang.org/doc/install ( > 1.4 version)

.. code-block:: shell

  go get github.com/tools/godep
  go get github.com/golang/lint/golint
  go get github.com/coreos/etcd
  go get golang.org/x/tools/cmd/cover
  go get golang.org/x/tools/cmd/vet
  make
  make install


How to install webui
-----------------------------------

+ install apache2
+ git clone https://github.com/cloudwan/gohan-webui
+ update config.json
  -> specify gohan server

  you need to setup auth_url and url based on your env

.. code-block:: javascript

  {
      "auth_url": "http://localhost:9090/v2.0",
      "gohan": {
          "gohan_schema": "/schema/gohan.json",
          "schema": "/gohan/v0.1/schemas",
          "url": "http://localhost:9090"
      },
      "navigation": [],
      "project_name": "Gohan UI",
      "routes": {
          "": {
              "controller": "GohanController.main"
          },
          "/login": {
              "controller": "GohanController.login"
          }
      },
      "sidebar": [
          [
              {
                  "path": "#",
                  "title": "Overview"
              }
          ]
      ],
      "templates": {
          "login": "bower_components/gohan/templates/login.html",
          "navbar": "bower_components/gohan/templates/navbar.html",
          "table": "bower_components/gohan/templates/table.html"
      },
      "title": "title"
  }



For more information, please take a look gohan-webui README


Package version
------------------------------------

Download package.zip for your distribution.

you can run server using

.. code-block:: shell

  gohan server

you can see webui on
http://localhost:9091/webui

Run API Server
------------------------------------

gohan server --config-file etc/api_server_config.yaml




