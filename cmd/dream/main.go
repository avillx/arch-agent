package main

// func main() {

// 	// config
// 	configPath := flag.String("config", "config.toml", "path to config file")
// 	dataPath := flag.String("datadir", ".", "path to data directory")
// 	flag.Parse()

// 	config, err := config.Load(*configPath)
// 	if err != nil {
// 		slog.Error("bad config", "error", err)
// 		return
// 	}

// 	// logging
// 	logging.Set(config.Logging.Pretty, config.Logging.Level)

// 	dreamer := openaiadapter.NewDreamer(
// 		openai.NewClient(
// 			option.WithBaseURL(config.LLMS.Dreaming.OpenAIURL),
// 			option.WithAPIKey(config.LLMS.Dreaming.APIKey),
// 		),
// 		config.LLMS.Dreaming.Model,
// 		config.LLMS.Dreaming.ReasoningEffort,
// 		config.LLMS.Dreaming.ToolChoice,
// 		config.LLMS.Dreaming.TopP,
// 		config.LLMS.Dreaming.FrequencyPenalty,
// 		config.LLMS.Dreaming.PresencePenalty,
// 		config.LLMS.Dreaming.Temperature,
// 		config.LLMS.Dreaming.Extras,
// 	)

// 	absolutePath, _ := filepath.Abs(*dataPath)

// 	dailyActivityStore := filestorage.NewMDDailyActivityStore(absolutePath + "/memory/daily_logs")

// 	knowledgeExplorer := filestorage.NewKnowledgeExplorer(absolutePath + "/memory")

// 	dreamUC := dream.NewDreamUseCase(
// 		dreamer,
// 		dailyActivityStore,
// 		knowledgeExplorer.CreateRecivier(),
// 		knowledgeExplorer.Knowledges,
// 		&logging.UseCaseLogger{},
// 	)

// 	files, err := dailyActivityStore.AllUndreamedFiles()
// 	if err != nil {
// 		slog.Error("can't load activity files", "error", err)
// 	}

// 	for _, f := range files {

// 		data, err := dailyActivityStore.Loadfile(f)
// 		if err != nil {
// 			slog.Error("can't load activity files ", "file", f, "error", err)
// 			return
// 		}

// 		// date from filename 04.06.10.md -> 04.06.10
// 		date := strings.TrimSuffix(f, filepath.Ext(f))

// 		if err := dreamUC.Execute(context.Background(), "at "+date+":\n"+data); err != nil {
// 			slog.Error("dreaming end with error %e", "error", err)
// 			return
// 		}

// 		if err := dailyActivityStore.MarkFileAsDreamed(f); err != nil {
// 			slog.Error("dreaming end with error %e", "error", err)
// 			return
// 		}

// 	}

// 	log.Print("dreaming")
// }
