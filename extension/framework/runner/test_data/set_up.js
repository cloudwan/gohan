var SCHEMAS = ["./schema.yaml"];
var PATH = "/v1.0/networks";

var setUpCalled = false;

function setUp() {
  setUpCalled = true;
}

function testSetUp() {
  if (!setUpCalled) {
    Fail("setUp not called!")
  }
}
