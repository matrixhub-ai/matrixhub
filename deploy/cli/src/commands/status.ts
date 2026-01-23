import chalk from 'chalk';
import {
  doesContainerExist,
  isContainerRunning,
  getContainerStatus
} from '../utils/docker.js';

export async function statusCommand(options: any) {
  console.log(chalk.blue.bold('\n📊 MatrixHub Status\n'));

  const exists = await doesContainerExist(options.name);
  if (!exists) {
    console.log(chalk.yellow(`⚠️  Container "${options.name}" not found\n`));
    process.exit(0);
  }

  const running = await isContainerRunning(options.name);
  const status = await getContainerStatus(options.name);

  console.log(chalk.cyan('Container name:'), chalk.bold(options.name));
  console.log(chalk.cyan('Status:'), running ? chalk.green.bold('Running ✅') : chalk.red.bold('Stopped ❌'));
  console.log(chalk.cyan('Details:'), status);

  if (running) {
    console.log(chalk.cyan('\n📍 Access MatrixHub at:'), chalk.bold('http://localhost:9527'));
  }
  console.log();
}
