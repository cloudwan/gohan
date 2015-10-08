function populate_network_subnets(context, network) {
    subnets = gohan_db_list(context["transaction"], "subnet", {"network_id": network.id});
    network.subnets = _.map(subnets, function(subnet) {
        return subnet.id;
    });
};

gohan_register_handler("post_list_in_transaction", function(context) {
    var networks = context.response.networks;
    _.each(networks, function(network) {
        populate_network_subnets(context, network);
    });
});

gohan_register_handler("post_show_in_transaction", function(context) {
    var network = context.response;
    populate_network_subnets(context, network);
});

gohan_register_handler("post_create_in_transaction", function(context) {
    var network = context.response;
    var tx = context.transaction;
    network.status = "PENDING_CREATE";
    gohan_db_update(tx, "network", network);
});

gohan_register_handler("post_update_in_transaction", function(context) {
    var network = context.response;
    var tx = context.transaction;
    network.status = "PENDING_UPDATE";
    gohan_db_update(tx, "network", network);
});

// During delete, we can't mark it as PENDING_DELETE because gohan,
// deletes resource from db anyway. Currently, there is no way to
// Avoid it and return without error
