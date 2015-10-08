function ValidationException(message) {
    this.message = message;
    this.name = "ValidationException";

    this.toDict = function () {
        return {
            "name": this.name,
            "message": this.message
        };
    };
}
