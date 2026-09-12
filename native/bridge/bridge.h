#ifndef BERGAMOT_BRIDGE_H
#define BERGAMOT_BRIDGE_H

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/*
 * A thin C ABI around the Bergamot/Marian C++ translation engine so that the
 * Go application can drive it through CGo without ever seeing C++ types.
 *
 * All handles are opaque. A single bridge owns an AsyncService plus the state
 * required to make its asynchronous translation interface appear blocking.
 */

typedef struct BergamotBridge BergamotBridge;
typedef struct BergamotTranslationModel BergamotTranslationModel;

/* Create a bridge owning an AsyncService with the given worker/cache sizes. */
BergamotBridge *bergamot_bridge_create(size_t num_workers, size_t cache_size);

/* Destroy the bridge, joining any pending work. */
void bergamot_bridge_destroy(BergamotBridge *bridge);

/* Load a translation model from a Marian config file (e.g. config.intgemm8bitalpha.yml).
 * Returns NULL on failure; call bergamot_bridge_last_error() for details. */
BergamotTranslationModel *bergamot_bridge_load_model(BergamotBridge *bridge, const char *config_path);

/* Release a loaded model. */
void bergamot_bridge_unload_model(BergamotTranslationModel *model);

/* Blocking translation. On success *translated is set to a heap string owned by the
 * caller (release with bergamot_bridge_free_string) and 0 is returned. */
int bergamot_bridge_translate(BergamotBridge *bridge, BergamotTranslationModel *model,
                              const char *text, char **translated);

/* Blocking pivoting translation using two models (source -> pivot, pivot -> target). */
int bergamot_bridge_pivot(BergamotBridge *bridge, BergamotTranslationModel *source_model,
                          BergamotTranslationModel *target_model, const char *text,
                          char **translated);

/* Cancel a pending translation. Unblocks any in-progress translate call. */
void bergamot_bridge_cancel(BergamotBridge *bridge);

/* Release a string returned by translate/pivot. */
void bergamot_bridge_free_string(char *string);

/* Human-readable description of the last error. Valid until the next call into the bridge. */
const char *bergamot_bridge_last_error(BergamotBridge *bridge);

#ifdef __cplusplus
}
#endif

#endif
