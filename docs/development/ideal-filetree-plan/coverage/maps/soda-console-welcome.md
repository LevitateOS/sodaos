# Soda console welcome

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-d4e4d1750a74"></a>

<a id="rustsoda-console-welcomesrcmainrs-1"></a>

## [rust/soda-console-welcome/src/main.rs](../../../../../rust/soda-console-welcome/src/main.rs)

Source assertions/fixtures were inspected for mapping only; no test execution, runtime or installed proof claimed. Large-file ledger retains committed declaration/member spans rather than grouping methods into one whole-file unit; responsibility follows the verified concern/branch mapping, not declaration count.

| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 1–14 | Native console guidance from current commissioning/private origin state; declarations/fields: `main`, `run`, `capture`, `have_command`, `hostname`, `render_config` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 15–18 | Native console guidance from current commissioning/private origin state; declaration/member main; declarations/fields: `main` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 19–139 | Native console guidance from current commissioning/private origin state; declaration/member run; declarations/fields: `run` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 140–157 | Native console guidance from current commissioning/private origin state; declaration/member capture; declarations/fields: `capture` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 158–169 | Native console guidance from current commissioning/private origin state; declaration/member have_command; declarations/fields: `have_command` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 170–183 | Native console guidance from current commissioning/private origin state; declaration/member hostname; declarations/fields: `hostname` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 184–244 | Native console guidance from current commissioning/private origin state; declaration/member render_config; declarations/fields: `render_config` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 245–263 | Bounded host status JSON parsing; declarations/fields: `parse_top_object` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 264 | Bounded host status JSON parsing; declaration/member JsonParser; declarations/fields: `JsonParser` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 265 | Bounded host status JSON parsing; declaration/member JsonParser.bytes; declarations/fields: `JsonParser.bytes` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 266 | Bounded host status JSON parsing; declaration/member JsonParser.pos; declarations/fields: `JsonParser.pos` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 267–270 | Bounded host status JSON parsing; declaration/member JsonParser.depth; declarations/fields: `JsonParser.depth` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 271–274 | Bounded host status JSON parsing; declaration/member peek; declarations/fields: `peek` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 275–280 | Bounded host status JSON parsing; declaration/member skip_ws; declarations/fields: `skip_ws` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 281–289 | Bounded host status JSON parsing; declaration/member literal; declarations/fields: `literal` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 290–326 | Bounded host status JSON parsing; declaration/member parse_object_top; declarations/fields: `parse_object_top` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 327–336 | Bounded host status JSON parsing; declaration/member parse_value_top; declarations/fields: `parse_value_top` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 337–349 | Bounded host status JSON parsing; declaration/member parse_any; declarations/fields: `parse_any` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 350–424 | Bounded host status JSON parsing; declaration/member parse_any_inner; declarations/fields: `parse_any_inner` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 425–468 | Bounded host status JSON parsing; declaration/member parse_number; declarations/fields: `parse_number` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 469–555 | Bounded host status JSON parsing; declaration/member parse_string; declarations/fields: `parse_string` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 556–568 | Bounded host status JSON parsing; declaration/member parse_hex4; declarations/fields: `parse_hex4` |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 569–580 | Bounded host status JSON parsing; declaration/member hex_value; declarations/fields: `hex_value` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 581–602 | Native listener/private origin validation for console guidance; declarations/fields: `valid_listen` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 603–629 | Native listener/private origin validation for console guidance; declaration/member is_loopback_ipv4; declarations/fields: `is_loopback_ipv4` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 630–676 | Native listener/private origin validation for console guidance; declaration/member valid_origin; declarations/fields: `valid_origin` |
| [N01](../../slices/networking.md#n01-private-origins-tls-and-activation) / active | 677–679 | Native listener/private origin validation for console guidance; declaration/member split_host_port; declarations/fields: `split_host_port` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 680–700 | Console guidance source assertions; declarations/fields: `valid_url_port`, `tests`, `top_object_extracts_strings_last_wins`, `string_escapes_decode_like_python`, `listen_shapes`, `origin_shapes` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 701–708 | Console guidance source assertions; declaration/member valid_url_port; declarations/fields: `valid_url_port` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 709–712 | Console guidance source assertions; declaration/member tests; declarations/fields: `tests` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 713–724 | Console guidance source assertions; declaration/member top_object_extracts_strings_last_wins; declarations/fields: `top_object_extracts_strings_last_wins` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 725–737 | Console guidance source assertions; declaration/member string_escapes_decode_like_python; declarations/fields: `string_escapes_decode_like_python` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 738–760 | Console guidance source assertions; declaration/member listen_shapes; declarations/fields: `listen_shapes` |
| [O02](../../slices/operator-administration.md#o02-operator-identity-bootstrap) / active | 761–790 | Console guidance source assertions; declaration/member origin_shapes; declarations/fields: `origin_shapes` |

