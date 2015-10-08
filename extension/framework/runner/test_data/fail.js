var SCHEMAS = ["./schema.yaml"];
var PATH = "/v1.0/networks";

function testFail() {
  Fail("called %s", "testFail");
}

function testFailNoMessage() {
  Fail();
}
