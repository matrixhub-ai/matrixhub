# Matrixhub UI

A modern, lightweight web interface for Matrixhub.

## Tech Stack

This project is built with modern web technologies:

- **[React](https://react.dev/)** 19.2.0 - UI library
- **[TypeScript](https://www.typescriptlang.org/)** 5.9.3 - Type-safe JavaScript
- **[Vite](https://vitejs.dev/)** 7.3.1 - Build tool and development server
- **[React Router DOM](https://reactrouter.com/)** - Client-side routing
- **[React Markdown](https://remarkjs.github.io/react-markdown/)** - Markdown rendering
- **[React Syntax Highlighter](https://github.com/react-syntax-highlighter/react-syntax-highlighter)** - Code syntax highlighting
- **[React Icons](https://react-icons.github.io/react-icons/)** - Icon library

## Prerequisites

Before you begin, ensure you have the following installed:

- **Node.js** 20.x or higher
- **npm** (comes with Node.js) or your preferred package manager (yarn, pnpm)

You can check your Node.js version with:
```bash
node --version
```

## Quick Start

Get up and running with Matrixhub UI in minutes:

### 1. Install dependencies

```bash
npm install
```

### 2. Start the development server

```bash
npm run dev
```

The application will start on `http://localhost:5173` (or another port if 5173 is in use).

### 3. Build for production

```bash
npm run build
```

The optimized production build will be generated in the `dist` directory.

## Available Scripts

In the project directory, you can run:

### `npm run dev`

Runs the app in development mode with hot module replacement (HMR).  
Open [http://localhost:5173](http://localhost:5173) to view it in the browser.

### `npm run build`

Builds the app for production to the `dist` folder.  
The build is optimized and minified for best performance.

### `npm run preview`

Locally preview the production build.  
Run this after `npm run build` to test the production bundle.

### `npm run lint`

Runs ESLint to check code quality and style issues.

## Project Structure

```
web/
├── src/
│   ├── api/           # API client and utilities
│   ├── assets/        # Static assets (images, icons)
│   ├── components/    # React components
│   │   ├── BranchSelector.tsx
│   │   ├── Breadcrumb.tsx
│   │   ├── CommitList.tsx
│   │   ├── FileTree.tsx
│   │   ├── FileViewer.tsx
│   │   └── ReadmeViewer.tsx
│   ├── pages/         # Page components
│   │   ├── HomePage.tsx
│   │   ├── RepoPage.tsx
│   │   ├── BlobPage.tsx
│   │   └── QueuePage.tsx
│   ├── utils/         # Utility functions
│   ├── App.tsx        # Main app component
│   └── main.tsx       # Entry point
├── public/            # Public static files
├── index.html         # HTML template
├── vite.config.ts     # Vite configuration
└── package.json       # Project dependencies
```

## Development

This project uses Vite for fast development and optimized builds. The development server features:

- ⚡️ Lightning-fast Hot Module Replacement (HMR)
- 🔧 Built-in TypeScript support
- 📦 Optimized bundling for production
- 🎯 ESLint integration for code quality

## License

See the main [Matrixhub repository](https://github.com/matrixhub-ai/matrixhub) for license information.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
