var SCHEMAS = ["./schema.yaml"];
var PATH = "/v1.0/networks";

function testExtension1Loaded() {
  from_extension1();
}

function testExtension2NotLoaded() {
  if (typeof from_extension2 !== "undefined") {
    Fail("Extension 2 should not be loaded!");
  }
}
