import chalk from 'chalk';
import ora from 'ora';
import {
  isContainerRunning,
  doesContainerExist,
  stopContainer
} from '../utils/docker.js';

export async function stopCommand(options: any) {
  console.log(chalk.blue.bold('\n🛑 Stopping MatrixHub\n'));

  const exists = await doesContainerExist(options.name);
  if (!exists) {
    console.log(chalk.yellow(`⚠️  Container "${options.name}" not found\n`));
    process.exit(0);
  }

  const running = await isContainerRunning(options.name);
  if (!running) {
    console.log(chalk.yellow(`⚠️  Container "${options.name}" is not running\n`));
    process.exit(0);
  }

  const spinner = ora('Stopping container...').start();
  try {
    await stopContainer(options.name);
    spinner.succeed('MatrixHub container stopped');
    console.log(chalk.gray('\n💡 Use "matrixhub start" to start it again\n'));
  } catch (error: any) {
    spinner.fail('Failed to stop container');
    console.error(chalk.red('\nError:'), error.message);
    process.exit(1);
  }
}
