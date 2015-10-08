var SCHEMAS = ["../neutron-v2.yaml"];
var PATH = "/v2.0/ports";

var authResponse = {
  "Access": {
    "ServiceCatalog": [
      {
        "Endpoints": [],
        "Name": "keystone"
      },
      {
        "Endpoints": [
          {
            "InternalURL": "http://127.0.0.1:5000/v2.0/nova"
          }
        ],
        "Name": "nova",
        "Type": "compute"
      }
    ]
  }
};

var portID = "65c0ee9f-d634-4522-8954-51021b570b0d";
var deviceID = "5e3898d7-11be-483e-9732-b2f5eccd2b2e";

function testNotifyPortStatusAllValues() {
  var states = [
    PORT_STATUS_ACTIVE,
    PORT_STATUS_DOWN,
    PORT_STATUS_ERROR,
    PORT_STATUS_BUILD,
    PORT_STATUS_NO_VALUE
  ];
  var port = {
    "id": portID,
    "device_id": deviceID,
    "device_owner": "compute:"
  };
  var prevPort = _.extend({}, port);
  _.each(states, function(prev) {
    _.each(states, function(curr) {
      port.status = curr;
      prevPort.status = prev;
      recordPortStatusChangedHelper(curr, prev, port);
      GohanTrigger("post_update_in_transaction", getContext(port, prevPort));
    });
  });
}

function testPortWithoutDeviceOwnerNoNotify() {
  var port = {
    "id": portID,
    "device_id": deviceID,
    "status": PORT_STATUS_ACTIVE
  };
  var prevPort = port;
  prevPort.status = PORT_STATUS_NO_VALUE;
  recordPortStatusChangedHelper(PORT_STATUS_ACTIVE, PORT_STATUS_NO_VALUE, port);
  GohanTrigger("post_update_in_transaction", getContext(port, prevPort));
}

function testPortWithoutDeviceIDNoNotify() {
  var port = {
    "id": portID,
    "device_owner": "compute:",
    "status": PORT_STATUS_ACTIVE
  };
  var prevPort = port;
  prevPort.status = PORT_STATUS_NO_VALUE;
  recordPortStatusChangedHelper(PORT_STATUS_ACTIVE, PORT_STATUS_NO_VALUE, port);
  GohanTrigger("post_update_in_transaction", getContext(port, prevPort));
}

function testNonComputeInstancesNoNotify() {
  var port = {
    "id": portID,
    "device_id": deviceID,
    "device_owner": "network:dhcp",
    "status": PORT_STATUS_ACTIVE
  };
  var prevPort = port;
  prevPort.status = PORT_STATUS_NO_VALUE;
  recordPortStatusChangedHelper(PORT_STATUS_ACTIVE, PORT_STATUS_NO_VALUE, port);
  GohanTrigger("post_update_in_transaction", getContext(port, prevPort));
}

function testComputeNotFoundNoNotify() {
  var port = {
    "id": portID,
    "device_id": deviceID,
    "device_owner": "compute:",
    "status": PORT_STATUS_ACTIVE
  };
  var prevPort = port;
  prevPort.status = PORT_STATUS_NO_VALUE;
  var context = {
    "response": _.extend({}, port),
    "resource": _.extend({}, prevPort)
  };
  GohanTrigger("post_update_in_transaction", context);
  if (!context.response_code || context.response_code != 503) {
    Fail("Wrong error response code: actual %s, expected 503", context.response_code);
  }
  var errorMessage = "Compute service could not be found.";
  if (!context.response || context.response != errorMessage) {
    Fail("Wrong error message: actual '%s', expected '%s'", context.response, errorMessage);
  }
}

function recordPortStatusChangedHelper(currentPortStatus, previousPortStatus, port) {
  if (!port.device_id || !port.device_owner || !isComputePort(port)) {
    return;
  }
  var eventName;
  if (previousPortStatus === PORT_STATUS_ACTIVE && currentPortStatus === PORT_STATUS_DOWN) {
    // port unplugged, i.e. 'network-vif-unplugged' event, not yet supported
    eventName = VIF_UNPLUGGED;
    return;
  } else if ([PORT_STATUS_NO_VALUE, PORT_STATUS_DOWN, PORT_STATUS_BUILD].indexOf(previousPortStatus) > -1 &&
      [PORT_STATUS_ACTIVE, PORT_STATUS_ERROR].indexOf(currentPortStatus) > -1) {
    eventName = VIF_PLUGGED;
  } else {
    return;
  }

  var portEvent = getEvent(port, eventName, currentPortStatus);
  var portEventResponse = _.extend({}, portEvent, {"code": 200});
  gohan_http.Expect("POST", "http://127.0.0.1:5000/v2.0/nova/os-server-external-events", {}, portEvent)
    .Return(portEventResponse);
}

function getContext(port, prevPort) {
  return {
    "auth": _.extend({}, authResponse),
    "response": _.extend({}, port),
    "resource": _.extend({}, prevPort)
  };
}

