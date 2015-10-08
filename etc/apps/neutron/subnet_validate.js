gohan_register_handler("pre_create", function(context) {
    var resource = context.resource;
    try {
        validateIpAddress(resource.gateway_ip, resource.ip_version);
        if (resource.dns_nameservers) {
          var dnsLength = resource.dns_nameservers.length;
          for(var i = 0; i < dnsLength; ++i) {
              validateIpAddress(resource.dns_nameservers[i], resource.ip_version);
          }
        }
        if (resource.host_routes) {
          var routesLength = resource.host_routes.length;
          for(i = 0; i < routesLength; ++i) {
              validateIpAddress(resource.host_routes[i]["destination"], resource.ip_version);
              validateIpAddress(resource.host_routes[i]["nexthop"], resource.ip_version);
          }
        }
    } catch (e) {
        if(e instanceof ValidationException) {
            context.response_code = 400;
            context.response = e.toDict();
        } else {
            throw e;
        }
    }
});
