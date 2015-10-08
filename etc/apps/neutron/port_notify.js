var PORT_STATUS_ACTIVE = "ACTIVE";
var PORT_STATUS_DOWN = "DOWN";
var PORT_STATUS_BUILD = "BUILD";
var PORT_STATUS_ERROR = "ERROR";
var PORT_STATUS_NO_VALUE = "NO_VALUE";

var NEUTRON_COMPUTE_EVENT_STATUS_MAP = {};
NEUTRON_COMPUTE_EVENT_STATUS_MAP[PORT_STATUS_ACTIVE] = 'completed';
NEUTRON_COMPUTE_EVENT_STATUS_MAP[PORT_STATUS_ERROR] = 'failed';
NEUTRON_COMPUTE_EVENT_STATUS_MAP[PORT_STATUS_DOWN] = 'completed';

var VIF_UNPLUGGED = "network-vif-unplugged";
var VIF_PLUGGED = "network-vif-plugged";

var notifyPath = "/os-server-external-events";

function recordPortStatusChanged(port, currentPortStatus, previousPortStatus, computeURL) {
  if (!port.device_id || !isComputePort(port)) {
    return;
  }
  var eventName;
  if (previousPortStatus === PORT_STATUS_ACTIVE && currentPortStatus === PORT_STATUS_DOWN) {
    eventName = VIF_UNPLUGGED;
    return;
  } else if ([PORT_STATUS_NO_VALUE, PORT_STATUS_DOWN, PORT_STATUS_BUILD].indexOf(previousPortStatus) > -1 &&
      [PORT_STATUS_ACTIVE, PORT_STATUS_ERROR].indexOf(currentPortStatus) > -1) {
    eventName = VIF_PLUGGED;
  } else {
    return;
  }

  gohan_http("POST", computeURL + notifyPath, {}, getEvent(port, eventName, currentPortStatus));
}

function isComputePort(port) {
  return port.device_id && port.device_owner && port.device_owner.lastIndexOf("compute:", 0) === 0;
}

function getEvent(port, eventName, currentPortStatus) {
  return {
    "server_uuid": port.device_id,
    "name": eventName,
    "status": NEUTRON_COMPUTE_EVENT_STATUS_MAP[currentPortStatus],
    "tag": port.id
  };
}

function getComputeURL(context) {
  var services;
  try {
    services = context.auth.Access.ServiceCatalog;
  } catch (e) {
    if (e instanceof TypeError) {
      context.response_code = 503;
      context.response = "Compute service could not be found.";
    } else {
      throw e;
    }
  }
  var computeURL = "";
  _.each(services, function(service) {
    if (service.Type === "compute") {
      computeURL = service.Endpoints[0].InternalURL;
    }
  });
  return "";
}

