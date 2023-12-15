#!/bin/sh

# Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
#
# SPDX-License-Identifier: MIT

index_html='/usr/share/nginx/html/index.html'

cat << EOF > ${index_html}
<!DOCTYPE html>
<html>
  <head>
    <title>Add-on Environment Variables</title>
    <style>
      html { color-scheme: light dark; }
      body { width: 35em; margin: 0 auto; font-family: Tahoma, Verdana, Arial, sans-serif; }
      pre  { font-size: large; }
    </style>
  </head>
  <body>
    <h1>Add-on Environment Variables</h1>
    <pre>
EOF

printenv >> ${index_html}

cat << EOF >> ${index_html}
    </pre>
  </body>
</html>
EOF

exit 0
