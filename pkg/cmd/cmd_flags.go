// Copyright 2023 The KCL Authors. All rights reserved.
// This file provides all the flags in the kpm cli.
//
// Deprecated: The entire contents of this file will be deprecated.
// Please use the kcl cli - https://github.com/kcl-lang/cli.

package cmd

const FLAG_INPUT = "input"
const FLAG_VENDOR = "vendor"
const FLAG_UPDATE = "update"
const FLAG_TAG = "tag"
const FLAG_TAR_PATH = "tar_path"

const FLAG_SETTING = "setting"
const FLAG_DISABLE_NONE = "disable_none"
const FLAG_ARGUMENT = "argument"
const FLAG_OVERRIDES = "overrides"
const FLAG_SORT_KEYS = "sort_keys"

const FLAG_QUIET = "quiet"
const FLAG_NO_SUM_CHECK = "no_sum_check"

// FLAG_NO_CACHE disables the persistent cache for the primary
// remote package referenced by `kpm run oci://...` (and the git
// equivalent). When set, every invocation re-downloads the
// referenced package instead of reusing the KPM cache.
//
// Equivalent to setting $KPM_RUN_NO_CACHE=1 or calling
// `client.WithRunNoCache()` programmatically.
const FLAG_NO_CACHE = "no_cache"
