# envreplace

simple utility to replace text with value from env

all text begin with `__ENV_` will be replaced with the env value

all text begin with `__ENVXML_` will be replaced with the env value but the value is using xml encoded (e.g. `a<b` will be replaced with `a&lt;b`)

all text begin with `__ENVJSON_` will be replaced with the env value but the value is using json encoded string, double quoted

all text begin with `__ENVJSONC_` will be replaced with like `__ENVJSON_` but surounding double quote is discared
