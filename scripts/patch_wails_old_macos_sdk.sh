#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.25.11}"
export GOCACHE="${GOCACHE:-$ROOT/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT/.cache/go-mod}"

cd "$ROOT"
go mod download github.com/wailsapp/wails/v3

MODULE_DIR="$(go list -m -f '{{.Dir}}' github.com/wailsapp/wails/v3)"
TARGET="$MODULE_DIR/pkg/application/autostart_darwin_smappservice.go"

if [[ ! -f "$TARGET" ]]; then
  echo "Wails source not found at $TARGET; run go mod download first." >&2
  exit 1
fi

if grep -q "NSClassFromString(@\"SMAppService\")" "$TARGET"; then
  echo "Wails old macOS SDK patch already applied"
  exit 0
fi

chmod u+w "$TARGET"
python3 - "$TARGET" <<'PY'
from pathlib import Path
import sys

target = Path(sys.argv[1])
text = target.read_text()
start = text.index("/*\n#cgo CFLAGS:")
end = text.index("*/\nimport \"C\"", start) + len("*/\nimport \"C\"")

replacement = r'''/*
#cgo CFLAGS: -mmacosx-version-min=10.15 -x objective-c -Wno-unguarded-availability-new
#cgo LDFLAGS: -framework Foundation -framework ServiceManagement

#include <stdlib.h> // free
#include <string.h> // strdup
#import <Foundation/Foundation.h>
#import <ServiceManagement/ServiceManagement.h>

// Return codes shared with the Go side.
enum {
	SMAS_OK              = 0,
	SMAS_UNAVAILABLE     = 1, // SMAppService class not present (pre macOS 13)
	SMAS_NOT_REGISTERED  = 2, // unregister called when nothing was registered
	SMAS_REQUIRES_APPROVAL = 3, // user disabled it in System Settings
	SMAS_ERROR           = 4, // generic failure; *outMsg populated
};

static id smAppServiceMain(void) {
	Class cls = NSClassFromString(@"SMAppService");
	if (cls == nil) {
		return nil;
	}
	SEL selector = NSSelectorFromString(@"mainAppService");
	if (![cls respondsToSelector:selector]) {
		return nil;
	}
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Warc-performSelector-leaks"
	return [cls performSelector:selector];
#pragma clang diagnostic pop
}

static BOOL smInvokeBool(id target, SEL selector, NSError **err) {
	NSMethodSignature *sig = [target methodSignatureForSelector:selector];
	NSInvocation *inv = [NSInvocation invocationWithMethodSignature:sig];
	[inv setTarget:target];
	[inv setSelector:selector];
	[inv setArgument:err atIndex:2];
	[inv invoke];
	BOOL ok = NO;
	[inv getReturnValue:&ok];
	return ok;
}

static NSInteger smInvokeInteger(id target, SEL selector) {
	NSMethodSignature *sig = [target methodSignatureForSelector:selector];
	NSInvocation *inv = [NSInvocation invocationWithMethodSignature:sig];
	[inv setTarget:target];
	[inv setSelector:selector];
	[inv invoke];
	NSInteger value = 0;
	[inv getReturnValue:&value];
	return value;
}

static int smAppServiceRegister(char** outMsg) {
	if (@available(macOS 13.0, *)) {
		@autoreleasepool {
			id svc = smAppServiceMain();
			if (svc == nil) {
				return SMAS_UNAVAILABLE;
			}
			SEL selector = NSSelectorFromString(@"registerAndReturnError:");
			if (![svc respondsToSelector:selector]) {
				return SMAS_UNAVAILABLE;
			}
			NSError *err = nil;
			if (smInvokeBool(svc, selector, &err)) {
				return SMAS_OK;
			}
			if (err != nil) {
				*outMsg = strdup([[err localizedDescription] UTF8String]);
			}
			return SMAS_ERROR;
		}
	}
	return SMAS_UNAVAILABLE;
}

static int smAppServiceUnregister(char** outMsg) {
	if (@available(macOS 13.0, *)) {
		@autoreleasepool {
			id svc = smAppServiceMain();
			if (svc == nil) {
				return SMAS_UNAVAILABLE;
			}
			SEL statusSelector = NSSelectorFromString(@"status");
			if ([svc respondsToSelector:statusSelector]) {
				NSInteger status = smInvokeInteger(svc, statusSelector);
				// SMAppServiceStatusNotRegistered = 0, NotFound = 3.
				if (status == 0 || status == 3) {
					return SMAS_NOT_REGISTERED;
				}
			}
			SEL selector = NSSelectorFromString(@"unregisterAndReturnError:");
			if (![svc respondsToSelector:selector]) {
				return SMAS_UNAVAILABLE;
			}
			NSError *err = nil;
			if (smInvokeBool(svc, selector, &err)) {
				return SMAS_OK;
			}
			if (err != nil) {
				*outMsg = strdup([[err localizedDescription] UTF8String]);
			}
			return SMAS_ERROR;
		}
	}
	return SMAS_UNAVAILABLE;
}

// smAppServiceStatus: 0 = unavailable, 1 = not registered / not found,
//                     2 = enabled, 3 = requires approval.
static int smAppServiceStatus(void) {
	if (@available(macOS 13.0, *)) {
		@autoreleasepool {
			id svc = smAppServiceMain();
			if (svc == nil) {
				return 0;
			}
			SEL selector = NSSelectorFromString(@"status");
			if (![svc respondsToSelector:selector]) {
				return 0;
			}
			NSInteger status = smInvokeInteger(svc, selector);
			// SMAppServiceStatusEnabled = 1, RequiresApproval = 2.
			if (status == 1) {
				return 2;
			}
			if (status == 2) {
				return 3;
			}
			return 1;
		}
	}
	return 0;
}
*/
import "C"'''

target.write_text(text[:start] + replacement + text[end:])
PY

echo "Patched Wails SMAppService calls for old macOS SDK builds: $TARGET"
