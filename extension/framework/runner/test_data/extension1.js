function from_extension1() {
}

gohan_register_handler("post_list", function(context) {
  context.triggered = true;
});

function CustomError(message) {
  this.message = message;
}
CustomError.prototype = new Error;

gohan_register_handler("pre_list", function(context) {
  throw new CustomError("ExtensionError");
});
