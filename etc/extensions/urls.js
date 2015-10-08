function substitutePrefix(namespacePrefix, schema) {
  schema.prefix = gohan_schema_url(schema.id);
}

gohan_register_handler("post_list", function(context) {
  for (var i = 0; i < context.response.schemas.length; ++i) {
    substitutePrefix(context.namespace_prefix, context.response.schemas[i]);
  }
});

gohan_register_handler("post_show", function(context) {
  substitutePrefix(context.namespace_prefix, context.response);
});

