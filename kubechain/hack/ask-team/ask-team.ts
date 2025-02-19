import { humanlayer } from "humanlayer";

const hl = humanlayer({
  runId: "ask-team",
  agentName: "kubechain-dev",
});

const askTeam = hl.humanAsTool({
  slack: {
    channel_or_user_id: "C08B5Q63E00",
    experimental_slack_blocks: true,
    context_about_channel_or_user: "a notebook channel",
  },
});

async function main() {
  const question = process.argv.slice(2).join(" ");
  if (!question) {
    console.error("USAGE: ask-team '<question>'");
    process.exit(1);
  }

  try {
    const response = await askTeam({ message: question });
    console.log("\nResponse from Team:", response);
  } catch (error) {
    console.error("Error:", error);
    process.exit(1);
  }
}

main();
