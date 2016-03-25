==============
Installation
==============

How to install Gohan
------------------------------------

You can download Gohan binary for your platfrom from
github release page.

https://github.com/cloudwan/gohan/releases

We have ansible role for Gohan for production usecases

https://github.com/cloudwan/ansible-gohan

Getting started using Simple pack
------------------------------------

(Step1) Download "Simple pack" which has Gohan binary, WebUI and Sample configuraion from https://github.com/cloudwan/ansible-gohan/releases

(Step2) Start server

.. code-block:: shell

  ./gohan server --config-file etc/gohan.yaml


(Step3) Open WebUI

https://localhost:9443/webui/

login with this ID/Password

ID: admin
Password: gohan

Now you can see sample application webui is running.
You can also access schema editing webui by adding "?type=metaschema" on URL.

https://localhost:9443/webui/?type=metaschema

Install from source / Dev env
------------------------------------

You need go (>1.4) to build gohan.

.. code-block:: shell

  # clone gohan
  git clone https://github.com/cloudwan/gohan.git
  cd gohan
  make deps # only the first time
  make
  make install