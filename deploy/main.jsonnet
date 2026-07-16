local image = std.extVar('image');
local namespace = if std.extVar('namespace') != '' then std.extVar('namespace') else 'orb-operator-system';
local profiles = std.extVar('profiles');

local api = import 'lib/api.libsonnet';
local controller = import 'lib/controller.libsonnet';

{
  apiVersion: 'v1',
  kind: 'List',
  items: api.generate() + controller.generate(image, namespace, profiles),
}
