import chalk from 'chalk';
import ora from 'ora';
import prompts from 'prompts';
import {
  doesContainerExist,
  isContainerRunning,
  stopContainer,
  removeContainer,
  pullImage,
  runContainer,
  getContainerStatus
} from '../utils/docker.js';

export async function updateCommand(options: any) {
  console.log(chalk.blue.bold('\n🔄 Updating MatrixHub\n'));

  const exists = await doesContainerExist(options.name);
  if (!exists) {
    console.log(chalk.yellow(`⚠️  Container "${options.name}" not found\n`));
    process.exit(0);
  }

  const running = await isContainerRunning(options.name);

  const response = await prompts({
    type: 'confirm',
    name: 'confirm',
    message: running 
      ? 'This will stop the current container and start a new one with the latest image. Continue?'
      : 'This will remove the current container and start a new one with the latest image. Continue?',
    initial: true
  });

  if (!response.confirm) {
    console.log(chalk.yellow('\nUpdate cancelled\n'));
    process.exit(0);
  }

  // Pull latest image
  const pullSpinner = ora(`Pulling latest image: ${options.image}...`).start();
  try {
    await pullImage(options.image);
    pullSpinner.succeed('Latest image pulled');
  } catch (error: any) {
    pullSpinner.fail('Failed to pull image');
    console.error(chalk.red('\nError:'), error.message);
    process.exit(1);
  }

  // Stop container if running
  if (running) {
    const stopSpinner = ora('Stopping container...').start();
    try {
      await stopContainer(options.name);
      stopSpinner.succeed('Container stopped');
    } catch (error: any) {
      stopSpinner.fail('Failed to stop container');
      console.error(chalk.red('\nError:'), error.message);
      process.exit(1);
    }
  }

  // Remove old container
  const removeSpinner = ora('Removing old container...').start();
  try {
    await removeContainer(options.name);
    removeSpinner.succeed('Old container removed');
  } catch (error: any) {
    removeSpinner.fail('Failed to remove container');
    console.error(chalk.red('\nError:'), error.message);
    process.exit(1);
  }

  console.log(chalk.green.bold('\n✅ MatrixHub updated successfully!'));
  console.log(chalk.gray('\n💡 Use "matrixhub start" to start the updated version\n'));
}
