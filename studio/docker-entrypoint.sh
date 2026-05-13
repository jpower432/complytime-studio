#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Render runtime config from environment variables.
envsubst < /usr/share/nginx/html/env.js.template > /usr/share/nginx/html/env.js
