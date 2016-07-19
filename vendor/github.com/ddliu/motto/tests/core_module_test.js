var fs = require('fs');

var data = fs.readFileSync('tests/data.json');
data = JSON.parse(data);
module.exports = data[0].name;