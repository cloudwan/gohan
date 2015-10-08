function ValidationError(message) {
  this.message = message;
}

ValidationError.prototype = new Error;

function isUUID(input) {
  return typeof input === "string";
}
