var SCHEMAS = ["./schema.yaml"];
var PATH = "/v1.0/networks";

function testMockExpectNotSpecified() {
  gohan_http.Return("OK");
}

function testMockReturnNotSpecified() {
  gohan_http.Expect("POST");
}

function testMockExpectEmpty() {
  gohan_http.Expect().Return("OK");
}

function testMockReturnEmpty() {
  gohan_http.Expect("POST").Return();
}
