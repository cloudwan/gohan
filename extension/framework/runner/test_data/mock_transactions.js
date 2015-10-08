var SCHEMAS = ["./schema.yaml"];
var PATH = "/v1.0/networks";

var network1 = {
  "id": "abc",
  "name": "net",
  "tenant_id": "tenant",
  "shared": false,
  "admin_state_up": false
};

function testMockTransactions() {
  var transaction = MockTransaction();
  var resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) != JSON.stringify([])) {
    Fail("Invalid resources - expected an empty array");
  }

  resp = gohan_db_create(transaction, "network", network1);

  resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) !== JSON.stringify([network1])) {
    Fail("Invalid resources in first transaction - expected %s, received %s",
        JSON.stringify([network1]), JSON.stringify(resources));
  }

  CommitMockTransaction();

  transaction = MockTransaction();

  resources = gohan_db_list(transaction, "network", {});
  if (JSON.stringify(resources) !== JSON.stringify([network1])) {
    Fail("Invalid resources in second transaction - expected %s, received %s",
        JSON.stringify([network1]), JSON.stringify(resources));
  }
}
