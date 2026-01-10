import { spawn, type SpawnOptions } from 'node:child_process'
import { readFile, writeFile, mkdir } from 'node:fs/promises'
import { join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import { stringify as yamlStringify, parse as yamlParse } from 'yaml'

const rootDir = fileURLToPath(new URL('..', import.meta.url))
const schemaPath = join(rootDir, 'docs/schema/websocket-api.yaml')
const outDir = join(rootDir, 'docs/generated-openapi')
const htmlOutputPath = join(outDir, 'openapi.html')
const redocCommand = 'pnpm'
const redocArgs = ['exec', 'redocly', 'build-docs'] as const

const runCommand = async (command: string, args: string[], options: SpawnOptions = {}) => {
    await new Promise<void>((resolve, reject) => {
        const child = spawn(command, args, options)
        child.on('error', reject)
        child.on('exit', (code, signal) => {
            if (code === 0) {
                resolve()
                return
            }
            reject(
                new Error(
                    `Command failed: ${command} ${args.join(' ')}${signal ? ` (signal: ${signal})` : ''}`
                )
            )
        })
    })
}

const buildHtmlFromSchema = async (inputPath: string, htmlPath: string) => {
    const args = [...redocArgs, inputPath, '-o', htmlPath]
    console.error(`Building Redoc HTML via: ${redocCommand} ${args.join(' ')}`)
    await runCommand(redocCommand, args, {
        stdio: 'inherit',
        env: {
            ...process.env,
        },
    })
    console.error(`Redoc HTML saved to ${relative(process.cwd(), htmlPath)}`)
}

const main = async () => {
    try {
        // Create output directory
        await mkdir(outDir, { recursive: true })

        // Read the existing OpenAPI schema
        console.error(`Reading OpenAPI schema from ${relative(process.cwd(), schemaPath)}`)
        const schemaContent = await readFile(schemaPath, 'utf8')

        // Parse and re-stringify to ensure valid YAML
        const schemaObj = yamlParse(schemaContent)

        // Generate Redoc HTML
        await buildHtmlFromSchema(schemaPath, htmlOutputPath)

        console.error('\n✅ Redoc generation complete!')
        console.error(`   View the documentation: ${relative(process.cwd(), htmlOutputPath)}`)
    } catch (error) {
        console.error('Failed to generate Redoc documentation:', error)
        process.exitCode = 1
    }
}

main().catch((error) => {
    console.error('Unexpected error:', error)
    process.exitCode = 1
})
