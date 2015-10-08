var SCHEMAS = ["./schema.yaml"];
var PATH = "/v1.0/networks";

function testExtensionError() {
  try {
    GohanTrigger("pre_list", {});
    Fail("Expected a CustomError exception.");
  } catch(e) {
    if (!(e instanceof CustomError)) {
       Fail("Unexpected error %s", JSON.stringify(e));
    }
  }
}
