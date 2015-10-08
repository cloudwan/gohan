var SCHEMAS = ["./schema.yaml"];
var PATH = "/v2.0/networks";

var context;
var network1;
var network2;

function setUp() {
  network1 = {
    'id': 'abc',
    'name': "Network 1",
    'tenant_id': 'azerty'
  };
  network2 = {
    'id': 'xyz',
    'name': "Network 2",
    'tenant_id': 'azerty'
  };

  network1 = gohan_db_create(MockTransaction(), "network", network1);
  network2 = gohan_db_create(MockTransaction(), "network", network2);

  context = {
    'schema': { /* ... */ },
    'http_request': { /* ... */ },
    'http_response': { /* ... */ },
    'path': '/gohan/v0.1/schema',
    'response': {
      'networks': [network1, network2]
    }
  }
}

function tearDown() {
  gohan_db_delete(MockTransaction(), "network", network1.id);
  gohan_db_delete(MockTransaction(), "network", network2.id);
}

// testNetworkListFilter tests whether network list is properly filtered
function testNetworkListFilter() {
  // next call to gohan_http will return 'foo'
  gohan_http.Expect("GET", "/unwanted/network", {}, null).Return(network2.id);
  // trigger post_list event on prepared context
  GohanTrigger('post_list', context);

  if (context.response.networks.length != 1) {
    Fail('Expected 1 network but %d networks found.',
        context.response.networks.length);
  }

  if (context.response.networks[0].id != network1.id) {
    Fail('Expected network with id "%s" but "%s" found.', network1.id,
        context.response.networks[0].id);
  }
}

function testSomethingElse() {
  gohan_http.Expect("GET", "/unwanted/network", {}, null).Return(undefined);
  try {
    GohanTrigger('post_list', context);
    Fail('Expected validation exception.');
  } catch(e) {
    // OK
  }
}
