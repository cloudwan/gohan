console.log("load sort.js");
var _ = require('./underscore.js');
function sort(data) {
    return _.sortBy(data, function(item) {
        return item.weight;
    });
}

function echo(msg) {
    console.log("[ECHO] " + msg)
}

exports.sort = sort;
exports.echo = echo;