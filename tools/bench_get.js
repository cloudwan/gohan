var benchrest = require('bench-rest');

var flow = {
    main: [
        { get: 'http://localhost:9091/v1.0/store/pets/', 
          headers: {"X-Auth-Token": "admin_token"} }
    ]
};

module.exports = flow;