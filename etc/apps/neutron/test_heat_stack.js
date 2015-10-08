var SCHEMAS = ["../neutron-v2.yaml"];
var PATH = "/v1.0/heat_stacks";

var network = {
  "id": "4e8e5957-649f-477b-9e5b-f1f75b21c03c",
  "name": "test_network",
  "status": "PENDING_CREATE",
  "tenant_id": "9bacb3c5d39d41a79512987f338cf177"
};

var network_stack = {
  "id": "network:4e8e5957-649f-477b-9e5b-f1f75b21c03c"
};

var port = {
  "status": "DOWN",
  "name": "private-port",
  "allowed_address_pairs": [],
  "admin_state_up": true,
  "network_id": "a87cc70a-3e15-4acf-8205-9b711a3531b7",
  "tenant_id": "d6700c0c9ffa4f1cb322cd4a1f3906fa",
  "device_owner": "compute:",
  "mac_address": "fa:16:3e:c9:cb:f0",
  "fixed_ips": [
    {
      "subnet_id": "a0304c3a-4f08-4c43-88af-d796509c97d2",
      "ip_address": "10.0.0.2"
    }
  ],
  "id": "65c0ee9f-d634-4522-8954-51021b570b0d",
  "security_groups": [
    "f0ac4394-7e4a-4409-9701-ba8be283dbc3"
  ],
  "device_id": "a0304c3a-4f08-4c43-88af-d796509c97d2"
};

var port_stack = {
  "id": "port:65c0ee9f-d634-4522-8954-51021b570b0d"
};

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

function create_network() {
  gohan_db_create(MockTransaction(), "network", network);
  var created_network = gohan_db_fetch(
      MockTransaction(), "network", network.id, network.tenant_id);
  if (!created_network) {
    Fail("Failed to create test network.");
  } else if (created_network.status !== "PENDING_CREATE") {
    Fail("Invalid initial network status: expected 'PENDING_CREATE' but found '%s'",
        created_network.status);
  }
}

function delete_network() {
  gohan_db_delete(MockTransaction(), "network", network.id);
}

function create_port() {
  gohan_db_create(MockTransaction(), "port", port);
  var created_port = gohan_db_fetch(
      MockTransaction(), "port", port.id, port.tenant_id);
  if (!created_port) {
    Fail("Failed to create test port.");
  } else if (created_port.status !== "DOWN") {
    Fail("Invalid initial port status: expected 'DOWN' but found '%s'",
        created_port.status);
  }
}

function delete_port() {
  gohan_db_delete(MockTransaction(), "port", port.id);
}

function testNoUndefinedNetworkStatus() {
  create_network();

  var portEvent = getEvent(port, VIF_PLUGGED, PORT_STATUS_ACTIVE);
  gohan_http.Expect("POST", "http://127.0.0.1:5000/v2.0/nova/os-server-external-events", {}, portEvent)
    .Return("");

  var stack = _.extend({"stack_status": "INVALID_STATUS"}, network_stack);
  var context = {
    "transaction": MockTransaction(),
    "response": stack
  };
  GohanTrigger("post_create_in_transaction", context);

  var updated_network = gohan_db_fetch(
      MockTransaction(), "network", network.id, network.tenant_id);
  if (!updated_network.status) {
    Fail("Invalid mapping for unknown heat stack status: '%s'",
        updated_network.status);
  }

  delete_network();
}

function testNetworkStatusUpdate() {
  create_network();

  var stack = _.extend({"stack_status": "CREATE_COMPLETE"}, network_stack);
  var context = {
    "transaction": MockTransaction(),
    "response": stack,
  };
  GohanTrigger("post_create_in_transaction", context);

  var updated_network = gohan_db_fetch(
      MockTransaction(), "network", network.id, network.tenant_id);
  if (updated_network.status !== "ACTIVE") {
    Fail("Invalid updated status: expected 'ACTIVE' but found '%s'",
        updated_network.status);
  }

  delete_network();
}

function testNoUndefinedPortStatus() {
  create_port();

  var stack = _.extend({"stack_status": "INVALID_STATUS"}, port_stack);
  var context = {
    "transaction": MockTransaction(),
    "response": stack,
    "auth": _.extend({}, authResponse)
  };
  GohanTrigger("post_create_in_transaction", context);

  var updated_port = gohan_db_fetch(
      MockTransaction(), "port", port.id, port.tenant_id);
  if (!updated_port.status) {
    Fail("Invalid mapping for unknown heat stack status: '%s'",
        updated_port.status);
  }

  delete_port();
}

function testPortStatusUpdate() {
  create_port();

  var portEvent = getEvent(port, VIF_PLUGGED, PORT_STATUS_ACTIVE);
  gohan_http.Expect("POST", "http://127.0.0.1:5000/v2.0/nova/os-server-external-events", {}, portEvent)
    .Return("");

  var stack = _.extend({"stack_status": "CREATE_COMPLETE"}, port_stack);
  var context = {
    "transaction": MockTransaction(),
    "response": stack,
    "auth": _.extend({}, authResponse)
  };
  GohanTrigger("post_create_in_transaction", context);

  var updated_port = gohan_db_fetch(
      MockTransaction(), "port", port.id, port.tenant_id);
  if (updated_port.status != "ACTIVE") {
    Fail("Invalid updated status: expected 'ACTIVE' but found '%s'",
        updated_port.status);
  }

  delete_port();
}

function testComputeNotFoundNoNotify() {
  create_port();

  var stack = _.extend({"stack_status": "CREATE_COMPLETE"}, port_stack);
  var context = {
    "transaction": MockTransaction(),
    "response": stack,
  };
  GohanTrigger("post_create_in_transaction", context);
  if (!context.response_code || context.response_code != 503) {
    Fail("Wrong error response code: actual %s, expected 503", context.response_code);
  }
  var errorMessage = "Compute service could not be found.";
  if (!context.response || context.response != errorMessage) {
    Fail("Wrong error message: actual '%s', expected '%s'", context.response, errorMessage);
  }
}
