var SCHEMAS = ["./schema.yaml", "./other_schema.yaml"];
var PATH = "/v1.0/networks";

var network = {
  "id": "4e8e5957-649f-477b-9e5b-f1f75b21c03c",
  "name": "Network 1",
  "tenant_id": "9bacb3c5d39d41a79512987f338cf177",
  "admin_state_up": false,
  "shared": false
};

var test_object = {
  "id": "4e8e5957-649f-477b-9e5b-f1f75b21c03c",
  "name": "Test Object 1",
  "tenant_id": "9bacb3c5d39d41a79512987f338cf177"
};

function testBothSchemasLoaded() {
  var result = gohan_db_create(MockTransaction(), "network", network);
  if (!_.isEqual(network, result)) {
    Fail("Failed to create an object using the first schema file.");
  }

  result = gohan_db_create(MockTransaction(), "test_object", test_object);
  if (!_.isEqual(test_object, result)) {
    Fail("Failed to create an object using the second schema file.");
  }
}
