import chalk from 'chalk';
import {
  doesContainerExist,
  getContainerLogs
} from '../utils/docker.js';

export async function logsCommand(options: any) {
  const exists = await doesContainerExist(options.name);
  if (!exists) {
    console.log(chalk.yellow(`\n⚠️  Container "${options.name}" not found\n`));
    process.exit(0);
  }

  if (options.follow) {
    console.log(chalk.gray(`Following logs for "${options.name}" (Ctrl+C to stop)...\n`));
  }

  try {
    await getContainerLogs(options.name, options.follow, options.tail);
  } catch (error: any) {
    console.error(chalk.red('\nError:'), error.message);
    process.exit(1);
  }
}
