var SCHEMAS = ["./schema.yaml"];
var PATH = "/v1.0/networks";

function testGohanBuiltins() {
  var transaction = MockTransaction();
  var network1 = {
      "id": "abc",
      "name": "net",
      "tenant_id": "tenant",
      "shared": false,
      "admin_state_up": false
  };
  var network2 = {
      "id": "def",
      "name": "nonet",
      "tenant_id": "tenant",
      "shared": false,
      "admin_state_up": false
  };

  var resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) != JSON.stringify([])) {
    Fail("Invalid resources - expected an empty array");
  }

  var resp = gohan_db_create(transaction, "network", network1);
  if (!_.isUndefined(resp.error)) {
    Fail("Failed to create a resource: %s", JSON.stringify(resp.error));
  }
  resp = gohan_db_create(transaction, "network", network2);
  if (!_.isUndefined(resp.error)) {
    Fail("Failed to create a resource: %s", JSON.stringify(resp.error));
  }

  resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) !== JSON.stringify([network1, network2])) {
    Fail("Invalid resources - expected %s, received %s",
        JSON.stringify([network1, network2]), JSON.stringify(resources));
  }

  network1.admin_state_up = true;
  gohan_db_update(transaction, "network", network1);
  var resource = gohan_db_fetch(transaction, "network", "abc", "tenant");

  if (JSON.stringify(resource) !== JSON.stringify(network1)) {
    FAIL("Invalid resource - expected %s, received %s",
        JSON.stringify(network1), JSON.stringify(resource));
  }

  gohan_db_delete(transaction, "network", network2.id);
  resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) !== JSON.stringify([network1])) {
    Fail("Invalid resources - expected %s, received %s",
        JSON.stringify([network1]), JSON.stringify(resources));
  }

  gohan_db_delete(transaction, "network", network1.id);
  resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) != JSON.stringify([])) {
    Fail("Invalid resources - expected an empty array");
  }
}
