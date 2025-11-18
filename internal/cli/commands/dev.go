// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/transire/transire/internal/cli/discovery"
	"github.com/transire/transire/pkg/transire"
)

// NewDevCommand creates the dev command group
func NewDevCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Development utilities",
		Long: `Development utilities for local testing and debugging.

Available subcommands:
  queues     - Queue management utilities
  schedules  - Schedule management utilities

These commands help you test your application locally by interacting
with the local development shims.`,
	}

	// Add subcommands
	cmd.AddCommand(newDevQueuesCommand())
	cmd.AddCommand(newDevSchedulesCommand())

	return cmd
}

// newDevQueuesCommand creates the dev queues command group
func newDevQueuesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queues",
		Short: "Queue development utilities",
		Long:  `Utilities for working with queues in local development.`,
	}

	// Add queue subcommands
	cmd.AddCommand(newDevQueuesListCommand())
	cmd.AddCommand(newDevQueuesSendCommand())

	return cmd
}

// newDevQueuesListCommand lists all registered queues
func newDevQueuesListCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered queues",
		Long:  `List all queues registered in the application.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			config, err := loadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Discover project
			project, err := discovery.DiscoverProject(".", config)
			if err != nil {
				return fmt.Errorf("failed to discover project: %w", err)
			}

			// Display registered queues
			fmt.Println("📋 Registered queues:")
			if len(project.QueueHandlers) == 0 {
				fmt.Println("  (No queue handlers registered)")
				fmt.Println("\n💡 Add queue handlers to your application and define them in transire.yaml")
				return nil
			}

			for _, handler := range project.QueueHandlers {
				timeout := handler.Config.VisibilityTimeoutSeconds
				batchSize := handler.Config.BatchSize
				maxRetries := handler.Config.MaxReceiveCount

				fmt.Printf("  • %s\n", handler.QueueName)
				fmt.Printf("    - batch_size: %d\n", batchSize)
				fmt.Printf("    - visibility_timeout: %ds\n", timeout)
				fmt.Printf("    - max_retries: %d\n", maxRetries)
				fmt.Printf("    - function: %s\n\n", handler.Function)
			}

			fmt.Printf("💡 Use 'transire dev queues send <queue> <message>' to test a queue\n")

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	return cmd
}

// newDevQueuesSendCommand sends a test message to a queue
func newDevQueuesSendCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "send <queue> <message>",
		Short: "Send a test message to a queue",
		Long: `Send a test message to a specific queue for local testing.

The message will be processed by the registered queue handler.

Examples:
  transire dev queues send email-queue '{"to":"test@example.com","subject":"Test"}'
  transire dev queues send notification-queue '{"user_id":123,"type":"welcome"}'`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			queueName := args[0]
			messageBody := args[1]

			// Load configuration
			config, err := loadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Discover project
			project, err := discovery.DiscoverProject(".", config)
			if err != nil {
				return fmt.Errorf("failed to discover project: %w", err)
			}

			// Find the queue handler
			var targetHandler *transire.QueueHandlerSpec
			for _, handler := range project.QueueHandlers {
				if handler.QueueName == queueName {
					targetHandler = &handler
					break
				}
			}

			if targetHandler == nil {
				fmt.Printf("❌ Queue '%s' not found\n", queueName)
				fmt.Println("\nAvailable queues:")
				for _, handler := range project.QueueHandlers {
					fmt.Printf("  • %s\n", handler.QueueName)
				}
				return fmt.Errorf("queue not found")
			}

			fmt.Printf("📤 Sending test message to queue '%s'\n", queueName)
			fmt.Printf("📝 Message: %s\n", messageBody)

			// Send message to the running application via dev API
			port := config.Development.HTTPPort
			if port == 0 {
				port = 3000
			}

			devAPIURL := fmt.Sprintf("http://localhost:%d/__dev/queues/send", port)

			payload := map[string]string{
				"queue_name": queueName,
				"message":    messageBody,
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("failed to marshal payload: %w", err)
			}

			resp, err := http.Post(devAPIURL, "application/json", bytes.NewBuffer(payloadJSON))
			if err != nil {
				return fmt.Errorf("failed to send message to dev API: %w (is your app running?)", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("dev API returned status %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			fmt.Printf("🔄 Message sent to running application\n")
			fmt.Printf("   Queue: %s\n", queueName)
			fmt.Printf("   Message ID: %s\n", result["message_id"])

			fmt.Printf("✅ Message processed (check application logs for details)\n")

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	return cmd
}

// newDevSchedulesCommand creates the dev schedules command group
func newDevSchedulesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedules",
		Short: "Schedule development utilities",
		Long:  `Utilities for working with scheduled tasks in local development.`,
	}

	// Add schedule subcommands
	cmd.AddCommand(newDevSchedulesListCommand())
	cmd.AddCommand(newDevSchedulesExecuteCommand())

	return cmd
}

// newDevSchedulesListCommand lists all registered schedules
func newDevSchedulesListCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered schedules",
		Long:  `List all scheduled tasks registered in the application.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			config, err := loadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Discover project
			project, err := discovery.DiscoverProject(".", config)
			if err != nil {
				return fmt.Errorf("failed to discover project: %w", err)
			}

			// Display registered schedules
			fmt.Println("📅 Registered schedules:")
			if len(project.ScheduleHandlers) == 0 {
				fmt.Println("  (No schedule handlers registered)")
				fmt.Println("\n💡 Add schedule handlers to your application and define them in transire.yaml")
				return nil
			}

			for _, handler := range project.ScheduleHandlers {
				enabled := handler.Config.Enabled
				timezone := handler.Config.Timezone
				timeout := handler.Config.TimeoutSeconds

				fmt.Printf("  • %s\n", handler.Name)
				fmt.Printf("    - schedule: %s\n", handler.Schedule)
				fmt.Printf("    - enabled: %t\n", enabled)
				fmt.Printf("    - timezone: %s\n", timezone)
				fmt.Printf("    - timeout: %ds\n", timeout)
				fmt.Printf("    - function: %s\n\n", handler.Function)
			}

			fmt.Printf("💡 Use 'transire dev schedules execute <schedule>' to test a schedule\n")

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	return cmd
}

// newDevSchedulesExecuteCommand executes a scheduled task manually
func newDevSchedulesExecuteCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "execute <schedule>",
		Short: "Execute a scheduled task manually",
		Long: `Execute a specific scheduled task manually for local testing.

This triggers the schedule handler immediately without waiting for the cron schedule.

Examples:
  transire dev schedules execute daily-cleanup
  transire dev schedules execute hourly-metrics`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scheduleName := args[0]

			// Load configuration
			config, err := loadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Discover project
			project, err := discovery.DiscoverProject(".", config)
			if err != nil {
				return fmt.Errorf("failed to discover project: %w", err)
			}

			// Find the schedule handler
			var targetHandler *transire.ScheduleHandlerSpec
			for _, handler := range project.ScheduleHandlers {
				if handler.Name == scheduleName {
					targetHandler = &handler
					break
				}
			}

			if targetHandler == nil {
				fmt.Printf("❌ Schedule '%s' not found\n", scheduleName)
				fmt.Println("\nAvailable schedules:")
				for _, handler := range project.ScheduleHandlers {
					fmt.Printf("  • %s\n", handler.Name)
				}
				return fmt.Errorf("schedule not found")
			}

			fmt.Printf("⏰ Executing schedule '%s'\n", scheduleName)
			fmt.Printf("📅 Schedule: %s\n", targetHandler.Schedule)
			fmt.Printf("🕒 Execution time: %s\n", time.Now().Format(time.RFC3339))

			// Send execution request to the running application via dev API
			port := config.Development.HTTPPort
			if port == 0 {
				port = 3000
			}

			devAPIURL := fmt.Sprintf("http://localhost:%d/__dev/schedules/execute", port)

			payload := map[string]string{
				"schedule_name": scheduleName,
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("failed to marshal payload: %w", err)
			}

			resp, err := http.Post(devAPIURL, "application/json", bytes.NewBuffer(payloadJSON))
			if err != nil {
				return fmt.Errorf("failed to execute schedule via dev API: %w (is your app running?)", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("dev API returned status %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			fmt.Printf("🔄 Schedule executed on running application\n")
			fmt.Printf("   Schedule: %s\n", scheduleName)
			fmt.Printf("   Event ID: %s\n", result["event_id"])

			fmt.Printf("✅ Schedule executed (check application logs for details)\n")

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	return cmd
}
