import chalk from 'chalk';
import ora from 'ora';
import {
  doesContainerExist,
  restartContainer
} from '../utils/docker.js';

export async function restartCommand(options: any) {
  console.log(chalk.blue.bold('\n🔄 Restarting MatrixHub\n'));

  const exists = await doesContainerExist(options.name);
  if (!exists) {
    console.log(chalk.yellow(`⚠️  Container "${options.name}" not found`));
    console.log(chalk.gray('💡 Use "matrixhub start" to create a new container\n'));
    process.exit(0);
  }

  const spinner = ora('Restarting container...').start();
  try {
    await restartContainer(options.name);
    spinner.succeed('MatrixHub container restarted');
    console.log(chalk.green.bold('\n✅ MatrixHub is now running!\n'));
  } catch (error: any) {
    spinner.fail('Failed to restart container');
    console.error(chalk.red('\nError:'), error.message);
    process.exit(1);
  }
}
