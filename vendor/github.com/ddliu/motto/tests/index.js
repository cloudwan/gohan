console.log("load index.js");
var data = require('./data');
var sort = require('./helper.js').sort;

return sort(data)[0].name;