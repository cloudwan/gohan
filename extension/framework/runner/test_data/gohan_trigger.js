var SCHEMAS = ["./schema.yaml"];
var PATH = "/v1.0/networks";

function testGohanTrigger() {
  context = {"triggered": false};
  GohanTrigger("post_list", context);
  if (!context.triggered) {
    Fail("'post_list' event not triggered!");
  }
}
