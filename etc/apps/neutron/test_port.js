var SCHEMAS = ["../neutron-v2.yaml"];
var PATH = "/v2.0/ports";

var port_stub = {
  "name": "private-port",
  "allowed_address_pairs": [],
  "admin_state_up": true,
  "network_id": "a87cc70a-3e15-4acf-8205-9b711a3531b7",
  "tenant_id": "d6700c0c9ffa4f1cb322cd4a1f3906fa",
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
  ]
};

var active_port_with_device_id = _.extend(
    {"status": "ACTIVE",
      "device_id": "5e3898d7-11be-483e-9732-b2f5eccd2b2e",
      "device_owner": "network:router_interface"},
    port_stub);

var active_port_without_device_id = _.extend(
    {"status": "ACTIVE",
      "device_id": "",
      "device_owner": ""},
    port_stub);

var inactive_port_with_device_id = _.extend(
    {"status": "DOWN",
      "device_id": "5e3898d7-11be-483e-9732-b2f5eccd2b2e",
      "device_owner": "network:router_interface"},
    port_stub);

function testActivePortWithDeviceId() {
  var port = _.extend({}, active_port_with_device_id);
  var context = {"response": {"port": port}};

  GohanTrigger("post_show", context)
  if (context.response.port.status !== "ACTIVE") {
    Fail("Expected port status 'ACTIVE', but '%v' found",
        context.response.port.status);
  }
}

function testActivePortWithoutDeviceId() {
  var port = _.extend({}, active_port_without_device_id);
  var context = {"response": {"port": port}};

  GohanTrigger("post_show", context)
  if (context.response.port.status !== "DOWN") {
    Fail("Expected port status 'DOWN', but '%v' found",
        context.response.port.status);
  }
}

function testInactivePortWithDeviceId() {
  var port = _.extend({}, inactive_port_with_device_id);
  var context = {"response": {"port": port}};

  GohanTrigger("post_show", context)
  if (context.response.port.status !== "DOWN") {
    Fail("Expected port status 'DOWN', but '%v' found",
        context.response.port.status);
  }
}

function testPortListResponse() {
  var ports = [
    _.extend({}, active_port_with_device_id),
    _.extend({}, active_port_without_device_id),
    _.extend({}, inactive_port_with_device_id)
  ];
  var context = {"response": {"ports": ports}};

  GohanTrigger("post_list", context);
  var expected = ["ACTIVE", "DOWN", "DOWN"];
  for (var i = 0; i < ports.length; i++) {
    if (context.response.ports[i].status != expected[i]) {
      Fail("Expected port status '%s' but '%s' found for port %v",
          expected[i], context.response.ports[i].status, i);
    }
  }
}
