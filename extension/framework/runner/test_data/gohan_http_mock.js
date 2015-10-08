var SCHEMAS = ["./schema.yaml"];
var PATH = "/v1.0/networks";

function testUnexpectedCall() {
  result = gohan_http("GET", "http://www.xyz.com", {}, {});
}

function testSingleReturnSingleCall() {
  var mock_response = "OK";
  gohan_http.Expect("GET", "http://www.xyz.com", {}, {}).Return(mock_response);
  var response = gohan_http("GET", "http://www.xyz.com", {}, {});
  if (response !== mock_response) {
    Fail();
  }
}

function testSingleReturnMultipleCalls() {
  var mock_response = "OK";
  gohan_http.Expect("GET", "http://www.xyz.com", {}, {}).Return(mock_response);
  var response = gohan_http("GET", "http://www.xyz.com", {}, {});
  if (response !== mock_response) {
    Fail();
  }

  var unexpected_response = gohan_http("GET", "http://www.xyz.com", {}, {});
}

function testWrongArgumentsCall() {
  var mock_response = "OK";
  gohan_http.Expect("POST", "http://www.abc.com", {"a": "a"}, {}).Return(mock_response);
  var response = gohan_http("GET", "http://www.xyz.com", {}, {});
  if (response !== mock_response) {
    Fail();
  }
}
