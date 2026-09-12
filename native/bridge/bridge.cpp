#include "bridge.h"

#include <condition_variable>
#include <cstring>
#include <memory>
#include <mutex>
#include <stdexcept>
#include <string>

#include "translator/parser.h"
#include "translator/response.h"
#include "translator/response_options.h"
#include "translator/service.h"
#include "translator/translation_model.h"

namespace {

using marian::bergamot::AsyncService;
using marian::bergamot::ResponseOptions;
using marian::bergamot::TranslationModel;

/*
 * State shared between a blocking translate call and the worker-thread callback
 * that completes it. Owned through a shared_ptr so it stays alive even if the
 * blocking caller is cancelled and returns before the callback fires.
 */
struct PendingTranslation {
  std::mutex mutex;
  std::condition_variable cv;
  std::string result;
  bool done = false;
  bool cancelled = false;
};

}  // namespace

struct BergamotBridge {
  std::unique_ptr<AsyncService> service;
  size_t num_workers = 1;

  std::mutex pending_mutex;
  std::shared_ptr<PendingTranslation> pending;

  std::string last_error;
};

struct BergamotTranslationModel {
  std::shared_ptr<TranslationModel> model;
};

extern "C" {

BergamotBridge *bergamot_bridge_create(size_t num_workers, size_t cache_size) {
  try {
    AsyncService::Config config;
    config.numWorkers = num_workers;
    config.cacheSize = cache_size;

    auto *bridge = new BergamotBridge();
    bridge->num_workers = num_workers;
    bridge->service = std::make_unique<AsyncService>(config);
    return bridge;
  } catch (const std::exception &error) {
    return nullptr;
  }
}

void bergamot_bridge_destroy(BergamotBridge *bridge) {
  if (bridge == nullptr)
    return;

  bergamot_bridge_cancel(bridge);
  delete bridge;
}

BergamotTranslationModel *bergamot_bridge_load_model(BergamotBridge *bridge,
                                                     const char *config_path) {
  if (bridge == nullptr || config_path == nullptr) {
    return nullptr;
  }

  try {
    auto options = marian::bergamot::parseOptionsFromFilePath(config_path);
    options->set("cpu-threads", static_cast<size_t>(bridge->num_workers), "mini-batch-words", 1000,
                 "alignment", "soft", "quiet", true);

    auto model = std::make_shared<TranslationModel>(options, bridge->num_workers);

    auto *handle = new BergamotTranslationModel();
    handle->model = model;
    return handle;
  } catch (const std::exception &error) {
    bridge->last_error = error.what();
    return nullptr;
  }
}

void bergamot_bridge_unload_model(BergamotTranslationModel *model) {
  delete model;
}

static int translate_with_models(BergamotBridge *bridge, BergamotTranslationModel *source_model,
                                 BergamotTranslationModel *target_model, const char *text,
                                 char **translated) {
  if (bridge == nullptr || source_model == nullptr || text == nullptr || translated == nullptr) {
    return 1;
  }

  auto pending = std::make_shared<PendingTranslation>();
  {
    std::lock_guard<std::mutex> lock(bridge->pending_mutex);
    bridge->pending = pending;
  }

  ResponseOptions options;
  options.alignment = false;
  options.qualityScores = false;
  options.HTML = false;

  auto callback = [pending](marian::bergamot::Response &&response) {
    std::lock_guard<std::mutex> lock(pending->mutex);
    pending->result = response.getTranslatedText();
    pending->done = true;
    pending->cv.notify_all();
  };

  try {
    if (target_model != nullptr) {
      bridge->service->pivot(source_model->model, target_model->model, std::string(text),
                             std::move(callback), options);
    } else {
      bridge->service->translate(source_model->model, std::string(text), std::move(callback),
                                 options);
    }
  } catch (const std::exception &error) {
    bridge->last_error = error.what();
    return 1;
  }

  std::unique_lock<std::mutex> lock(pending->mutex);
  pending->cv.wait(lock, [&pending] { return pending->done || pending->cancelled; });

  if (pending->done) {
    *translated = strdup(pending->result.c_str());
    return 0;
  }

  bridge->last_error = "translation cancelled";
  return 1;
}

int bergamot_bridge_translate(BergamotBridge *bridge, BergamotTranslationModel *model,
                              const char *text, char **translated) {
  return translate_with_models(bridge, model, nullptr, text, translated);
}

int bergamot_bridge_pivot(BergamotBridge *bridge, BergamotTranslationModel *source_model,
                          BergamotTranslationModel *target_model, const char *text,
                          char **translated) {
  return translate_with_models(bridge, source_model, target_model, text, translated);
}

void bergamot_bridge_cancel(BergamotBridge *bridge) {
  if (bridge == nullptr)
    return;

  bridge->service->clear();

  std::shared_ptr<PendingTranslation> pending;
  {
    std::lock_guard<std::mutex> lock(bridge->pending_mutex);
    pending = bridge->pending;
  }

  if (pending != nullptr) {
    std::lock_guard<std::mutex> lock(pending->mutex);
    pending->cancelled = true;
    pending->cv.notify_all();
  }
}

void bergamot_bridge_free_string(char *string) {
  free(string);
}

const char *bergamot_bridge_last_error(BergamotBridge *bridge) {
  if (bridge == nullptr)
    return "null bridge";
  return bridge->last_error.c_str();
}

}  // extern "C"
