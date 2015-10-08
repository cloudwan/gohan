function randomMac(prefix) {
  var mac = prefix || '54:52:00';

  for (var i = 0; i < 6; i++) {
    if (i%2 === 0) mac += ':';
    mac += Math.floor(Math.random()*16).toString(16);
  }

  return mac;
};

gohan_register_handler("pre_create", function(context) {
    var port = context.resource;
    if (!port["mac_address"]) {
        port["mac_address"] = randomMac()
    }
});

function updatePortStatus(port) {
  if (port.status == "ACTIVE" && port.device_id == "") {
    port.status = "DOWN";
  }
}

gohan_register_handler("post_list", function(context) {
  for (var i = 0; i < context.response.ports.length; i++) {
    updatePortStatus(context.response.ports[i]);
  }
});

gohan_register_handler("post_show", function(context) {
  updatePortStatus(context.response.port);
});

gohan_register_handler("post_update_in_transaction", function(context) {
  var port = context.response;
  var prevPort = context.resource;
  var computeURL = getComputeURL(context);
  if (computeURL === "") {
    return;
  }
  recordPortStatusChanged(port, port.status, prevPort.status, computeURL);
});
