var benchrest = require('bench-rest');

var flow = {
    main: [
        { post: 'http://localhost:9091/v1.0/store/pets/', 
          headers: {"X-Auth-Token": "admin_token"},
          json: {
              "pet": {
              id: "cat",
              name: "cat",
              tenant_id: "admin",
              status: "available"}
          } 
        }
    ]
};

module.exports = flow;