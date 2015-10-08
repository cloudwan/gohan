function map_network_status(stack_status) {
  var stack_status_mapping = {
    "CREATE_IN_PROGRESS": "PENDING_CREATE",
    "UPDATE_IN_PROGRESS": "PENDING_UPDATE",
    "DELETE_IN_PROGRESS": "PENDING_DELETE",

    "INIT_COMPLETE": "ACTIVE",

    "CREATE_COMPLETE": "ACTIVE",
    "UPDATE_COMPLETE": "ACTIVE",
    "DELETE_COMPLETE": "DELETED",

    "CREATE_ERROR": "ERROR",
    "UPDATE_ERROR": "ERROR",
    "DELETE_ERROR": "ERROR",
  };

  if (stack_status_mapping.hasOwnProperty(stack_status)) {
    return stack_status_mapping[stack_status];
  }

  return "ERROR";
}

function map_port_status(stack_status) {
  var port_status = {
    "ACTIVE": 'ACTIVE',
    "BUILD": 'BUILD',
    "DOWN": 'DOWN',
    "ERROR": 'ERROR',
    "NO_VALUE": 'NO_VALUE'
  };
  var stack_status_mapping = {
    "CREATE_IN_PROGRESS": port_status.BUILD,
    "UPDATE_IN_PROGRESS": port_status.BUILD,
    "DELETE_IN_PROGRESS": port_status.DOWN,

    "INIT_COMPLETE": port_status.ACTIVE,

    "CREATE_COMPLETE": port_status.ACTIVE,
    "UPDATE_COMPLETE": port_status.ACTIVE,
    "DELETE_COMPLETE": port_status.NO_VALUE,

    "CREATE_ERROR": port_status.ERROR,
    "UPDATE_ERROR": port_status.ERROR,
    "DELETE_ERROR": port_status.ERROR
  };

  if (stack_status_mapping.hasOwnProperty(stack_status)) {
    return stack_status_mapping[stack_status];
  }

  return port_status.NO_VALUE;
}

function update_resource_state(context) {
    var RESOURCES = ["network", "port"];
    var MAPPERS = {
      "network": map_network_status,
      "port": map_port_status
    };

    var heat_stack = context.response;
    var tx = context.transaction;
    var parts = heat_stack.id.split(":");
    var resource_type = parts[0];
    var resource_id = parts[1];

    if (RESOURCES.indexOf(resource_type) >= 0) {
        var resource = gohan_db_fetch(tx, resource_type, resource_id, "");
        var previousStatus = resource.status;
        var currentStatus = MAPPERS[resource_type](heat_stack.stack_status);
        resource.status = currentStatus;
        gohan_db_update(tx, resource_type, resource);
        if (resource_type === "port") {
          var computeURL = getComputeURL(context);
          if (computeURL === "") {
            return;
          }
          recordPortStatusChanged(resource, currentStatus, previousStatus, computeURL);
        }
    }
}

gohan_register_handler("post_create_in_transaction", function(context) {
    update_resource_state(context);
});

gohan_register_handler("post_update_in_transaction", function(context) {
    update_resource_state(context);
});
