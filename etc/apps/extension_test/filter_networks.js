// filterNetworks removes the network with the unwanted id
gohan_register_handler("post_list", function filterNetworks(context) {
    unwanted = gohan_http("GET", "/unwanted/network", {}, null);

    if (!isUUID(unwanted)) {
      throw new ValidationError("Invalid unwanted network's ID.");
    }

    for (var i = 0; i < context.response.networks.length; i++) {
        if (context.response.networks[i].id === unwanted) {
            context.response.networks.splice(i, 1);
            break;
        }
    }
});
