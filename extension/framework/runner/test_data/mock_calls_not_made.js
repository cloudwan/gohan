var SCHEMAS = ["./schema.yaml"];
var PATH = "/v1.0/networks";

function testFirstMockCallNotMade() {
  gohan_http.Expect("POST").Return("OK");
}

function testLastMockCallNotMade() {
  gohan_http.Expect("POST").Return("OK");
  gohan_http.Expect("GET").Return("OK");
  gohan_http("POST");
}
