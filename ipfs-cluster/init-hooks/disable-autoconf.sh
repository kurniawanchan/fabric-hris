#!/bin/sh
# Runs via kubo's own /container-init.d hook mechanism, AFTER ipfs init,
# as the correct "ipfs" user with correct permissions already applied by
# the image's own entrypoint script — avoids the permission problems of
# bypassing that script entirely (found the hard way).
#
# kubo 0.43 ships a NEW "AutoConf" system: several config fields default to
# the literal string "auto", resolved to real values at runtime by AutoConf
# — Bootstrap/DNS.Resolvers/Routing.DelegatedRouters/Ipns.DelegatedPublishers.
# Simply disabling AutoConf (needed on a private swarm — see the compose
# file's own comment) leaves those "auto" placeholders meaningless and the
# daemon refuses to start. For a PRIVATE cluster we do not want the public
# bootstrap/resolver/router defaults anyway, so clear all four explicitly
# rather than resolve them — found empirically, this exact combination is
# what the real 0.43 image actually requires, not documented anywhere this
# project's pinned corpus covers (that corpus is Fabric-only).
ipfs config AutoConf.Enabled --json false
ipfs bootstrap rm --all
ipfs config Bootstrap --json '[]'
ipfs config DNS.Resolvers --json '{}'
ipfs config Routing.DelegatedRouters --json '[]'
ipfs config Ipns.DelegatedPublishers --json '[]'
