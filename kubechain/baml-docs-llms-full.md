# 🏠 Welcome

## Motivation

Prompts are more than just f-strings; they're actual functions with logic that can quickly become complex to organize, maintain, and test.

Currently, developers craft LLM prompts as if they're writing raw HTML and CSS in text files, lacking:

* Type safety
* Hot-reloading or previews
* Linting

The situation worsens when dealing with structured outputs. Since most prompts rely on Python and Pydantic, developers must *execute* their code and set up an entire Python environment just to test a minor prompt adjustment, or they have to setup a whole Python microservice just to call an LLM.

BAML allows you to view and run prompts directly within your editor, similar to how Markdown Preview function -- no additional setup necessary, that interoperates with all your favorite languages and frameworks.

Just as TSX/JSX provided the ideal abstraction for web development, BAML offers the perfect abstraction for prompt engineering. Watch our [demo video](/guide/introduction/what-is-baml#demo-video) to see it in action.


<Steps>
  ### Write a BAML function definition

  ```baml main.baml
  class WeatherAPI {
    city string @description("the user's city")
    timeOfDay string @description("As an ISO8601 timestamp")
  }

  function UseTool(user_message: string) -> WeatherAPI {
    client "openai/gpt-4o"
    prompt #"
      Extract.... {# we will explain the rest in the guides #}
    "#
  }
  ```

  Here you can run tests in the VSCode Playground.

  ### Generate `baml_client` from those .baml files.

  This is auto-generated code with all boilerplate to call the LLM endpoint, parse the output, fix broken JSON, and handle errors.

  <img src="file:a53fb1a7-18d1-4e6b-b418-302320355d80" />

  ### Call your function in any language

  with type-safety, autocomplete, retry-logic, robust JSON parsing, etc..

  <CodeGroup>
    ```python Python
    import asyncio
    from baml_client import b
    from baml_client.types import WeatherAPI

    def main():
        weather_info = b.UseTool("What's the weather like in San Francisco?")
        print(weather_info)
        assert isinstance(weather_info, WeatherAPI)
        print(f"City: {weather_info.city}")
        print(f"Time of Day: {weather_info.timeOfDay}")

    if __name__ == '__main__':
        main()
    ```

    ```typescript TypeScript
    import { b } from './baml_client'
    import { WeatherAPI } from './baml_client/types'
    import assert from 'assert'

    const main = async () => {
      const weatherInfo = await b.UseTool("What's the weather like in San Francisco?")
      console.log(weatherInfo)
      assert(weatherInfo instanceof WeatherAPI)
      console.log(`City: ${weatherInfo.city}`)
      console.log(`Time of Day: ${weatherInfo.timeOfDay}`)
    }
    ```

    ```ruby Ruby
    require_relative "baml_client/client"

    $b = Baml.Client

    def main
      weather_info = $b.UseTool(user_message: "What's the weather like in San Francisco?")
      puts weather_info
      raise unless weather_info.is_a?(Baml::Types::WeatherAPI)
      puts "City: #{weather_info.city}"
      puts "Time of Day: #{weather_info.timeOfDay}"
    end
    ```

    ```python Other Languages
    # read the installation guide for other languages!
    ```
  </CodeGroup>
</Steps>

Continue on to the [Installation Guides](/guide/installation-language) for your language to setup BAML in a few minutes!

You don't need to migrate 100% of your LLM code to BAML in one go! It works along-side any existing LLM framework.


# What is baml_src?

**baml\_src** is where you keep all your BAML files, and where all the prompt-related code lives. It must be named `baml_src` for our tooling to pick it up, but it can live wherever you want.

It helps keep your project organized, and makes it easy to separate prompt engineering from the rest of your code.

<img src="file:a53fb1a7-18d1-4e6b-b418-302320355d80" />

Some things to note:

1. All declarations within this directory are accessible across all files contained in the `baml_src` folder.
2. You can have multiple files, and even nest subdirectories.

You don't need to worry about including this directory when deploying your code. See: [Deploying](/guide/development/deploying/aws)


# What is baml_client?

**baml\_client** is the code that gets generated from your BAML files that transforms your BAML prompts into the same equivalent function in your language, with validated type-safe outputs.

<img src="file:a53fb1a7-18d1-4e6b-b418-302320355d80" />

```python Python
from baml_client import b
resume_info = b.ExtractResume("....some text...")
```

This has all the boilerplate to:

1. call the LLM endpoint with the right parameters,
2. parse the output,
3. fix broken JSON (if any)
4. return the result in a nice typed object.
5. handle errors

In Python, your BAML types get converted to Pydantic models. In Typescript, they get converted to TypeScript types, and so on. **BAML acts like a universal type system that can be used in any language**.

### Generating baml\_client

Refer to the **[Installation](/guide/installation-language/python)** guides for how to set this up for your language, and how to generate it.

But at a high-level, you just include a [generator block](/ref/baml/generator) in any of your BAML files.

<CodeBlocks>
  ```baml Python
  generator target {
      // Valid values: "python/pydantic", "typescript", "ruby/sorbet"
      output_type "python/pydantic"

      // Where the generated code will be saved (relative to baml_src/)
      output_dir "../"

      // What interface you prefer to use for the generated code (sync/async)
      // Both are generated regardless of the choice, just modifies what is exported
      // at the top level
      default_client_mode "sync"

      // Version of runtime to generate code for (should match installed baml-py version)
      version "0.54.0"
  }
  ```

  ```baml TypeScript
  generator target {
      // Valid values: "python/pydantic", "typescript", "ruby/sorbet"
      output_type "typescript"

      // Where the generated code will be saved (relative to baml_src/)
      output_dir "../"

      // What interface you prefer to use for the generated code (sync/async)
      // Both are generated regardless of the choice, just modifies what is exported
      // at the top level
      default_client_mode "async"

      // Version of runtime to generate code for (should match the package @boundaryml/baml version)
      version "0.54.0"
  }
  ```

  ```baml Ruby (beta)
  generator target {
      // Valid values: "python/pydantic", "typescript", "ruby/sorbet"
      output_type "ruby/sorbet"

      // Where the generated code will be saved (relative to baml_src/)
      output_dir "../"

      // Version of runtime to generate code for (should match installed `baml` package version)
      version "0.54.0"
  }
  ```

  ```baml OpenAPI
  generator target {
      // Valid values: "python/pydantic", "typescript", "ruby/sorbet", "rest/openapi"
      output_type "rest/openapi"

      // Where the generated code will be saved (relative to baml_src/)
      output_dir "../"

      // Version of runtime to generate code for (should match installed `baml` package version)
      version "0.54.0"

      // 'baml-cli generate' will run this after generating openapi.yaml, to generate your OpenAPI client
      // This command will be run from within $output_dir
      on_generate "npx @openapitools/openapi-generator-cli generate -i openapi.yaml -g OPENAPI_CLIENT_TYPE -o ."
  }
  ```
</CodeBlocks>

The `baml_client` transforms a BAML function into the same equivalent function in your language,


# VSCode Extension

We provide a BAML VSCode extension:     [https://marketplace.visualstudio.com/items?itemName=Boundary.baml-extension](https://marketplace.visualstudio.com/items?itemName=Boundary.baml-extension)

| Feature                                                   | Supported |
| --------------------------------------------------------- | --------- |
| Syntax highlighting for BAML files                        | ✅         |
| Code snippets for BAML                                    | ✅         |
| LLM playground for testing BAML functions                 | ✅         |
| Jump to definition for BAML files                         | ✅         |
| Jump to definition between Python/TS files and BAML files | ✅         |
| Auto generate `baml_client` on save                       | ✅         |
| BAML formatter                                            | ❌         |

## Opening BAML Playground

Once you open a `.baml` file, in VSCode, you should see a small button over every BAML function: `Open Playground`.

<img src="file:e21aee8c-ce02-469d-bf13-32431b2c3d38" />

Or type `BAML Playground` in the VSCode Command Bar (`CMD + Shift + P` or `CTRL + Shift + P`) to open the playground.

<img src="file:17d133ab-cd27-4846-bb4c-89e6cacb24d1" />

## Setting Env Variables

Click on the `Settings` button in top right of the playground and set the environment variables.

It should have an indicator saying how many unset variables are there.

<img src="file:b23bad64-be98-445b-bce3-8385aefc1b99" />

The playground should persist the environment variables between closing and opening VSCode.

<Tip>
  You can set environment variables lazily. If anything is unset you'll get an error when you run the function.
</Tip>

<Info>
  Environment Variables are stored in VSCode's local storage! We don't save any additional data to disk, or send them across the network.
</Info>

## Running Tests

* Click on the `Run All Tests` button in the playground.

* Press the `▶️` button next to an individual test case to run that just that test case.

## Switching Functions

The playground will automatically switch to the function you're currently editing.

To manually change it, click on the current function name in the playground (next to the dropdown) and search for your desired function.

## Switching Test Cases

The test case with the highlighted background is the currently rendered test case. Clicking on a different test case will render that test case.

<img src="file:6d0137b7-c463-4ff4-9847-d60a15fdcad8" />

You can toggle between seeing the results of all test cases or all test cases for the current function.

<img src="file:211b83d8-14ec-4457-b630-59f620dd3876" />


# Cursor

Refer to the [Cursor Extension Installation Guide](https://www.cursor.com/how-to-install-extension) to install the extension in Cursor.

<Warning>
  You may need to update BAML extension manually using the process above. Auto-update does not seem to be working well for many extensions in Cursor.
</Warning>


# Typescript

<Note>
  You can check out this repo: 

  [https://github.com/BoundaryML/baml-examples/tree/main/nextjs-starter](https://github.com/BoundaryML/baml-examples/tree/main/nextjs-starter)
</Note>

To set up BAML with Typescript do the following:

<Steps>
  ### Install BAML VSCode/Cursor Extension

  [https://marketplace.visualstudio.com/items?itemName=boundary.baml-extension](https://marketplace.visualstudio.com/items?itemName=boundary.baml-extension)

  * syntax highlighting
  * testing playground
  * prompt previews

  ### Install BAML

  <CodeBlocks>
    ```bash npm
    npm install @boundaryml/baml
    ```

    ```bash pnpm
    pnpm add @boundaryml/baml
    ```

    ```bash yarn
    yarn add @boundaryml/baml
    ```

    ```bash deno
    deno install npm:@boundaryml/baml
    ```
  </CodeBlocks>

  ### Add BAML to your existing project

  This will give you some starter BAML code in a `baml_src` directory.

  <CodeBlocks>
    ```bash npm
    npx baml-cli init
    ```

    ```bash pnpm
    pnpm exec baml-cli init
    ```

    ```bash yarn
    yarn baml-cli init
    ```

    ```bash deno
    deno run -A npm:@boundaryml/baml/baml-cli init
    ```
  </CodeBlocks>

  ### Generate the `baml_client` typescript package from `.baml` files

  One of the files in your `baml_src` directory will have a [generator block](/ref/baml/generator). This tells BAML how to generate the `baml_client` directory, which will have auto-generated typescript code to call your BAML functions.

  ```bash
  npx baml-cli generate
  ```

  ```bash deno
  deno run -A npm:@boundaryml/baml/baml-cli generate
  ```

  You can modify your `package.json` so you have a helper prefix in front of your build command.

  ```json package.json
  {
    "scripts": {
      // Add a new command
      "baml-generate": "baml-cli generate",
      // Always call baml-generate on every build.
      "build": "npm run baml-generate && tsc --build",
    }
  }
  ```

  See [What is baml\_src](/guide/introduction/baml_src) to learn more about how this works.

  <img src="file:ecb48558-4c61-4bd6-9706-6f5e5bb34bb3" />

  <Tip>
    If you set up the [VSCode extension](https://marketplace.visualstudio.com/items?itemName=Boundary.baml-extension), it will automatically run `baml-cli generate` on saving a BAML file.
  </Tip>

  ### Use a BAML function in Typescript!

  <Error>
    If 

    `baml_client`

     doesn't exist, make sure to run the previous step! 
  </Error>

  <CodeBlocks>
    ```typescript index.ts
    import {b} from "baml_client"
    import type {Resume} from "baml_client/types"

    async function Example(raw_resume: string): Resume {
      // BAML's internal parser guarantees ExtractResume
      // to be always return a Resume type
      const response = await b.ExtractResume(raw_resume);
      return response;
    }

    async function ExampleStream(raw_resume: string): Resume {
      const stream = b.stream.ExtractResume(raw_resume);
      for await (const msg of stream) {
        console.log(msg) // This will be a Partial<Resume> type
      }

      // This is guaranteed to be a Resume type.
      return await stream.get_final_response();
    }
    ```

    ```typescript sync_example.ts
    import {b} from "baml_client/sync_client"
    import type {Resume} from "baml_client/types"

    function Example(raw_resume: string): Resume {
      // BAML's internal parser guarantees ExtractResume
      // to be always return a Resume type
      const response = b.ExtractResume(raw_resume);
      return response;
    }

    // Streaming is not available in the sync_client.

    ```
  </CodeBlocks>
</Steps>



# REST API (other languages)

<Info>
  Requires BAML version >=0.55
</Info>

<Warning>
  This feature is a preview feature and may change. Please provide feedback either
  in [Discord][discord] or on [GitHub][openapi-feedback-github-issue] so that
  we can stabilize the feature and keep you updated!
</Warning>

BAML allows you to expose your BAML functions as RESTful APIs:

<img src="file:89d51bba-21ba-4522-8102-de04d22a2498" />

We integrate with [OpenAPI](openapi) (universal API definitions), so you can get typesafe client libraries for free!

<Steps>
  ### Install BAML VSCode Extension

  [https://marketplace.visualstudio.com/items?itemName=boundary.baml-extension](https://marketplace.visualstudio.com/items?itemName=boundary.baml-extension)

  * syntax highlighting
  * testing playground
  * prompt previews

  ### Install NPX + OpenAPI

  <Tabs>
    <Tab title="macOS (brew)">
      ```bash
      brew install npm openapi-generator
      # 'npm' will install npx
      # 'openapi-generator' will install both Java and openapi-generator-cli
      ```
    </Tab>

    <Tab title="Linux (apt)">
      OpenAPI requires `default-jdk`

      ```bash
      apt install npm default-jdk -y
      # 'npm' will install npx; 'default-jdk' will install java
      ```
    </Tab>

    <Tab title="Linux (yum/dnf)">
      OpenAPI requires Java

      ```bash
      dnf install npm java-21-openjdk -y
      # dnf is the successor to yum
      ```

      Amazon Linux 2023:

      ```bash
      dnf install npm java-21-amazon-corretto -y
      # 'npm' will install npx
      # 'java-21-amazon-corretto' will install java
      ```

      Amazon Linux 2:

      ```bash
      curl -sL https://rpm.nodesource.com/setup_16.x | bash -
      yum install nodejs -y
      # 'nodejs' will install npx
      amazon-linux-extras install java-openjdk11 -y
      # 'java-openjdk11' will install java
      ```
    </Tab>

    <Tab title="Windows">
      To install `npx` and `java` (for OpenAPI):

      1. Use the [Node.js installer](https://nodejs.org/en/download/prebuilt-installer) to install `npx` (default installer settings are fine).
      2. Run `npm install -g npm@latest` to update `npx` (there is currently an [issue][npx-windows-issue] with the default install of `npx` on Windows where it doesn't work out of the box).
      3. Run the [Adoptium OpenJDK `.msi` installer](https://adoptium.net/temurin/releases/?os=windows) (install the JDK; default installer settings are fine).

      You can verify that `npx` and `java` are installed by running:

      ```powershell
      npx -version
      java -version
      ```
    </Tab>

    <Tab title="Other">
      To install `npx`, use the [Node.js installer](https://nodejs.org/en/download/prebuilt-installer).

      To install `java` (for OpenAPI), use the [Adoptium OpenJDK packages](https://adoptium.net/installation/linux/).
    </Tab>
  </Tabs>

  ### Add BAML to your existing project

  This will give you some starter BAML code in a `baml_src` directory.

  <Tabs>
    <Tab title="C#">
      ```bash
      npx @boundaryml/baml init \
        --client-type rest/openapi --openapi-client-type csharp
      ```
    </Tab>

    <Tab title="C++">
      <Tip>
        OpenAPI supports 

        [5 different C++ client types][openapi-client-types]

        ;
        any of them will work with BAML.
      </Tip>

      ```bash
      npx @boundaryml/baml init \
        --client-type rest/openapi --openapi-client-type cpp-restsdk
      ```
    </Tab>

    <Tab title="Go">
      ```bash
      npx @boundaryml/baml init \
        --client-type rest/openapi --openapi-client-type go
      ```
    </Tab>

    <Tab title="Java">
      ```bash
      npx @boundaryml/baml init \
        --client-type rest/openapi --openapi-client-type java
      ```

      Notice that `on_generate` has been initialized for you to:

      * run the OpenAPI generator to generate a Java client library, and *also*
      * run `mvn clean install` to install the generated client library to your
        local Maven repository

      <Warning>
        If you only use Maven through an IDE (e.g. IntelliJ IDEA), you should
        remove `&& mvn clean install` from the generated `on_generate` command.
      </Warning>
    </Tab>

    <Tab title="PHP">
      ```bash
      npx @boundaryml/baml init \
        --client-type rest/openapi --openapi-client-type php
      ```
    </Tab>

    <Tab title="Ruby">
      ```bash
      npx @boundaryml/baml init \
        --client-type rest/openapi --openapi-client-type ruby
      ```
    </Tab>

    <Tab title="Rust">
      ```bash
      npx @boundaryml/baml init \
        --client-type rest/openapi --openapi-client-type rust
      ```
    </Tab>

    <Tab title="Other">
      As long as there's an OpenAPI client generator that works with your stack,
      you can use it with BAML. Check out the [full list in the OpenAPI docs][openapi-client-types].

      ```bash
      npx @boundaryml/baml init \
        --client-type rest/openapi --openapi-client-type $OPENAPI_CLIENT_TYPE
      ```
    </Tab>
  </Tabs>

  ### Start the BAML development server

  ```bash
  npx @boundaryml/baml dev --preview
  ```

  This will do four things:

  * serve your BAML functions over a RESTful interface on `localhost:2024`
  * generate an OpenAPI schema in `baml_client/openapi.yaml`
  * run `openapi-generator -g $OPENAPI_CLIENT_TYPE` in `baml_client` directory to
    generate an OpenAPI client for you to use
  * re-run the above steps whenever you modify any `.baml` files

  <Note>
    BAML-over-REST is currently a preview feature. Please provide feedback
    either in [Discord][discord] or on [GitHub][openapi-feedback-github-issue]
    so that we can stabilize the feature and keep you updated!
  </Note>

  ### Use a BAML function in any language!

  `openapi-generator` will generate a `README` with instructions for installing
  and using your client; we've included snippets for some of the most popular
  languages below. Check out
  [`baml-examples`](https://github.com/BoundaryML/baml-examples) for example
  projects with instructions for running them.

  <Note>
    We've tested the below listed OpenAPI clients, but not all of them. If you run
    into issues with any of the OpenAPI clients, please let us know, either in
    [Discord][discord] or by commenting on
    [GitHub][openapi-feedback-github-issue] so that we can either help you out
    or fix it!
  </Note>

  <Tabs>
    <Tab title="Go">
      Run this with `go run main.go`:

      ```go main.go
      package main

      import (
      	"context"
      	"fmt"
      	"log"
        baml "my-golang-app/baml_client"
      )

      func main() {
      	cfg := baml.NewConfiguration()
      	b := baml.NewAPIClient(cfg).DefaultAPI
      	extractResumeRequest := baml.ExtractResumeRequest{
      		Resume: "Ada Lovelace (@gmail.com) was an English mathematician and writer",
      	}
      	resp, r, err := b.ExtractResume(context.Background()).ExtractResumeRequest(extractResumeRequest).Execute()
      	if err != nil {
      		fmt.Printf("Error when calling b.ExtractResume: %v\n", err)
      		fmt.Printf("Full HTTP response: %v\n", r)
      		return
      	}
      	log.Printf("Response from server: %v\n", resp)
      }
      ```
    </Tab>

    <Tab title="Java">
      First, add the OpenAPI-generated client to your project.

      <AccordionGroup>
        <Accordion title="If you have 'mvn' in your PATH">
          You can use the default `on_generate` command, which will tell `baml dev` to
          install the OpenAPI-generated client into your local Maven repository by running
          `mvn clean install` every time you save a change to a BAML file.

          To depend on the client in your local Maven repo, you can use these configs:

          <CodeGroup>
            ```xml pom.xml
            <dependency>
              <groupId>org.openapitools</groupId>
              <artifactId>openapi-java-client</artifactId>
              <version>0.1.0</version>
              <scope>compile</scope>
            </dependency>
            ```

            ```kotlin settings.gradle.kts
            repositories {
                mavenCentral()
                mavenLocal()
            }

            dependencies {
                implementation("org.openapitools:openapi-java-client:0.1.0")
            }
            ```
          </CodeGroup>
        </Accordion>

        <Accordion title="If you don't have 'mvn' in your PATH">
          You'll probably want to comment out `on_generate` and instead use either the [OpenAPI Maven plugin] or [OpenAPI Gradle plugin] to build your OpenAPI client.

          [OpenAPI Maven plugin]: https://github.com/OpenAPITools/openapi-generator/tree/master/modules/openapi-generator-maven-plugin

          [OpenAPI Gradle plugin]: https://github.com/OpenAPITools/openapi-generator/tree/master/modules/openapi-generator-gradle-plugin

          <CodeGroup>
            ```xml pom.xml
            <build>
                <plugins>
                    <plugin>
                        <groupId>org.openapitools</groupId>
                        <artifactId>openapi-generator-maven-plugin</artifactId>
                        <version>7.8.0</version> <!-- Use the latest stable version -->
                        <executions>
                            <execution>
                                <goals>
                                    <goal>generate</goal>
                                </goals>
                                <configuration>
                                    <inputSpec>${project.basedir}/baml_client/openapi.yaml</inputSpec>
                                    <generatorName>baml</generatorName> <!-- or another generator name, e.g. 'kotlin' or 'spring' -->
                                    <output>${project.build.directory}/generated-sources/openapi</output>
                                    <apiPackage>com.boundaryml.baml_client.api</apiPackage>
                                    <modelPackage>com.boundaryml.baml_client.model</modelPackage>
                                    <invokerPackage>com.boundaryml.baml_client</invokerPackage>
                                    <java8>true</java8>
                                </configuration>
                            </execution>
                        </executions>
                    </plugin>
                </plugins>
            </build>
            ```

            ```kotlin settings.gradle.kts
            plugins {
                id("org.openapi.generator") version "7.8.0"
            }

            openApiGenerate {
                generatorName.set("java") // Change to 'kotlin', 'spring', etc. if needed
                inputSpec.set("${projectDir}/baml_client/openapi.yaml")
                outputDir.set("$buildDir/generated-sources/openapi")
                apiPackage.set("com.boundaryml.baml_client.api")
                modelPackage.set("com.boundaryml.baml_client.model")
                invokerPackage.set("com.boundaryml.baml_client")
                additionalProperties.set(mapOf("java8" to "true"))
            }

            sourceSets["main"].java {
                srcDir("$buildDir/generated-sources/openapi/src/main/java")
            }

            tasks.named("compileJava") {
                dependsOn("openApiGenerate")
            }
            ```
          </CodeGroup>
        </Accordion>
      </AccordionGroup>

      Then, copy this code into wherever your `main` function is:

      ```Java
      import com.boundaryml.baml_client.ApiClient;
      import com.boundaryml.baml_client.ApiException;
      import com.boundaryml.baml_client.Configuration;
      // NOTE: baml_client/README.md will suggest importing from models.* - that is wrong.
      // See https://github.com/OpenAPITools/openapi-generator/issues/19431 for more details.
      import com.boundaryml.baml_client.model.*;
      import com.boundaryml.baml_client.api.DefaultApi;

      public class Example {
        public static void main(String[] args) {
          ApiClient defaultClient = Configuration.getDefaultApiClient();
          DefaultApi apiInstance = new DefaultApi(defaultClient);
          ExtractResumeRequest extractResumeRequest = new ExtractResumeRequest(); // ExtractResumeRequest | 
          try {
            Resume result = apiInstance.extractResume(extractResumeRequest);
            System.out.println(result);
          } catch (ApiException e) {
            System.err.println("Exception when calling DefaultApi#extractResume");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
          }
        }
      }

      ```
    </Tab>

    <Tab title="PHP">
      <Warning>
        The PHP OpenAPI generator doesn't support OpenAPI's `oneOf` type, which is
        what we map BAML union types to. Please let us know if this is an issue for
        you, and you need help working around it.
      </Warning>

      First, add the OpenAPI-generated client to your project:

      ```json composer.json
          "repositories": [
              {
                  "type": "path",
                  "url": "baml_client"
              }
          ],
          "require": {
              "boundaryml/baml-client": "*@dev"
          }
      ```

      You can now use this code to call a BAML function:

      ```PHP
      <?php
      require_once(__DIR__ . '/vendor/autoload.php');

      $apiInstance = new BamlClient\Api\DefaultApi(
          new GuzzleHttp\Client()
      );
      $extract_resume_request = new BamlClient\Model\ExtractResumeRequest();
      $extract_resume_request->setResume("Marie Curie was a Polish and naturalised-French physicist and chemist who conducted pioneering research on radioactivity");

      try {
          $result = $apiInstance->extractResume($extract_resume_request);
          print_r($result);
      } catch (Exception $e) {
          echo 'Exception when calling DefaultApi->extractResume: ', $e->getMessage(), PHP_EOL;
      }
      ```
    </Tab>

    <Tab title="Ruby">
      Use `ruby -Ilib/baml_client app.rb` to run this:

      ```ruby app.rb
      require 'baml_client'
      require 'pp'

      api_client = BamlClient::ApiClient.new
      b = BamlClient::DefaultApi.new(api_client)

      extract_resume_request = BamlClient::ExtractResumeRequest.new(
        resume: <<~RESUME
          John Doe

          Education
          - University of California, Berkeley
          - B.S. in Computer Science
          - graduated 2020

          Skills
          - Python
          - Java
          - C++
        RESUME
      )

      begin
        result = b.extract_resume(extract_resume_request)
        pp result

        edu0 = result.education[0]
        puts "Education: #{edu0.school}, #{edu0.degree}, #{edu0.year}"
      rescue BamlClient::ApiError => e
        puts "Error when calling DefaultApi#extract_resume"
        pp e
      end
      ```
    </Tab>

    <Tab title="Rust">
      <Tip>
        If you're using `cargo watch -- cargo build` and seeing build failures because it can't find
        the generated `baml_client`, try increasing the delay on `cargo watch` to 1 second like so:

        ```bash
        cargo watch --delay 1 -- cargo build
        ```
      </Tip>

      First, add the OpenAPI-generated client to your project:

      ```toml Cargo.toml
      [dependencies]
      baml-client = { path = "./baml_client" }
      ```

      You can now use `cargo run`:

      ```rust
      use baml_client::models::ExtractResumeRequest;
      use baml_client::apis::default_api as b;

      #[tokio::main]
      async fn main() {
          let config = baml_client::apis::configuration::Configuration::default();

          let resp = b::extract_resume(&config, ExtractResumeRequest {
              resume: "Tony Hoare is a British computer scientist who has made foundational contributions to programming languages, algorithms, operating systems, formal verification, and concurrent computing.".to_string(),
          }).await.unwrap();

          println!("{:#?}", resp);
      }
      ```
    </Tab>
  </Tabs>
</Steps>

[discord]: https://discord.gg/BTNBeXGuaS

[openapi-feedback-github-issue]: https://github.com/BoundaryML/baml/issues/892

[npx-windows-issue]: https://github.com/nodejs/node/issues/53538

[openapi-client-types]: https://github.com/OpenAPITools/openapi-generator#overview


## Dynamically setting LLM API Keys

You can set the API key for an LLM dynamically by passing in the key as a header or as a parameter (depending on the provider), using the [ClientRegistry](/guide/baml-advanced/client-registry).


# Docker

When you develop with BAML, the BAML VScode extension generates a `baml_client` directory (on every save) with all the generated code you need to use your AI functions in your application.

We recommend you add `baml_client` to your `.gitignore` file to avoid committing generated code to your repository, and re-generate the client code when you build and deploy your application.

You *could* commit the generated code if you're starting out to not deal with this, just make sure the VSCode extension version matches your baml package dependency version (e.g. `baml-py` for python and `@boundaryml/baml` for TS) so there are no compatibility issues.

To build your client you can use the following command. See also [baml-cli generate](/ref/baml-cli/generate):

<CodeBlocks>
  ```dockerfile python Dockerfile
  RUN baml-cli generate --from path-to-baml_src
  ```

  ```dockerfile TypeScript Dockerfile
  # Do this early on in the dockerfile script before transpiling to JS
  RUN npx baml-cli generate --from path-to-baml_src
  ```

  ```dockerfile Ruby Dockerfile
  RUN bundle add baml
  RUN bundle exec baml-cli generate --from path/to/baml_src
  ```
</CodeBlocks>


# OpenAPI

<Info>
  This feature was added in: v0.55.0.
</Info>

<Info>
  This page assumes you've gone through the [OpenAPI quickstart].
</Info>

[OpenAPI quickstart]: /docs/get-started/quickstart/openapi

To deploy BAML as a RESTful API, you'll need to do three things:

* host your BAML functions in a Docker container
* update your app to call it
* run BAML and your app side-by-side using `docker-compose`

Read on to learn how to do this with `docker-compose`.

<Tip>
  You can also run `baml-cli` in a subprocess from your app directly, and we
  may recommend this approach in the future. Please let us know if you'd
  like to see instructions for doing so, and in what language, by asking in
  [Discord][discord] or [on the GitHub issue][openapi-feedback-github-issue].
</Tip>

## Host your BAML functions in a Docker container

In the directory containing your `baml_src/` directory, create a
`baml.Dockerfile` to host your BAML functions in a Docker container:

<Note>
  BAML-over-HTTP is currently a preview feature. Please provide feedback either
  in [Discord][discord] or on [GitHub][openapi-feedback-github-issue] so that
  we can stabilize the feature and keep you updated!
</Note>

```docker title="baml.Dockerfile"
FROM node:20

WORKDIR /app
COPY baml_src/ .

# If you want to pin to a specific version (which we recommend):
# RUN npm install -g @boundaryml/baml@VERSION
RUN npm install -g @boundaryml/baml

CMD baml-cli serve --preview --port 2024
```

<Tabs>
  <Tab title="Using docker-compose">
    Assuming you intend to run your own application in a container, we recommend
    using `docker-compose` to run your app and BAML-over-HTTP side-by-side:

    ```bash
    docker compose up --build --force-recreate
    ```

    ```yaml title="docker-compose.yaml"
    services:
      baml-over-http:
        build:
          # This will build baml.Dockerfile when you run docker-compose up
          context: .
          dockerfile: baml.Dockerfile
        healthcheck:
          test: [ "CMD", "curl", "-f", "http://localhost:2024/_debug/ping" ]
          interval: 1s
          timeout: 100ms
          retries: 3
        # This allows you to 'curl localhost:2024/_debug/ping' from your machine,
        # i.e. the Docker host
        ports:
          - "2024:2024"

      debug-container:
        image: amazonlinux:latest
        depends_on:
          # Wait until the baml-over-http healthcheck passes to start this container
          baml-over-http:
            condition: service_healthy
        command: "curl -v http://baml-over-http:2024/_debug/ping"
    ```

    <Note>
      To call the BAML server from your laptop (i.e. the host machine), you must use
      `localhost:2024`. You may only reach it as `baml-over-http:2024` from within
      another Docker container.
    </Note>
  </Tab>

  <Tab title="Using docker">
    If you don't care about using `docker-compose`, you can just run:

    ```bash
    docker build -t baml-over-http -f baml.Dockerfile .
    docker run -p 2024:2024 baml-over-http
    ```
  </Tab>
</Tabs>

To verify for yourself that BAML-over-HTTP is up and running, you can run:

```bash
curl http://localhost:2024/_debug/ping
```

## Update your app to call it

Update your code to use `BOUNDARY_ENDPOINT`, if set, as the endpoint for your BAML functions.

If you plan to use [Boundary Cloud](/ref/cloud/functions/get-started) to host your BAML functions, you should also update it to use `BOUNDARY_API_KEY`.

<Tabs>
  <Tab title="Go">
    ```go
    import (
        "os"
        baml "my-golang-app/baml_client"
    )

    func main() {
        cfg := baml.NewConfiguration()
        if boundaryEndpoint := os.Getenv("BOUNDARY_ENDPOINT"); boundaryEndpoint != "" {
            cfg.BasePath = boundaryEndpoint
        }
        if boundaryApiKey := os.Getenv("BOUNDARY_API_KEY"); boundaryApiKey != "" {
            cfg.DefaultHeader["Authorization"] = "Bearer " + boundaryApiKey
        }
        b := baml.NewAPIClient(cfg).DefaultAPI
        // Use `b` to make API calls
    }
    ```
  </Tab>

  <Tab title="Java">
    ```java
    import com.boundaryml.baml_client.ApiClient;
    import com.boundaryml.baml_client.ApiException;
    import com.boundaryml.baml_client.Configuration;
    import com.boundaryml.baml_client.api.DefaultApi;
    import com.boundaryml.baml_client.auth.*;

    public class ApiExample {
        public static void main(String[] args) {
            ApiClient apiClient = Configuration.getDefaultApiClient();

            String boundaryEndpoint = System.getenv("BOUNDARY_ENDPOINT");
            if (boundaryEndpoint != null && !boundaryEndpoint.isEmpty()) {
                apiClient.setBasePath(boundaryEndpoint);
            }

            String boundaryApiKey = System.getenv("BOUNDARY_API_KEY");
            if (boundaryApiKey != null && !boundaryApiKey.isEmpty()) {
                apiClient.addDefaultHeader("Authorization", "Bearer " + boundaryApiKey);
            }

            DefaultApi apiInstance = new DefaultApi(apiClient);
            // Use `apiInstance` to make API calls
        }
    }
    ```
  </Tab>

  <Tab title="PHP">
    ```php
    require_once(__DIR__ . '/vendor/autoload.php');

    $config = BamlClient\Configuration::getDefaultConfiguration();

    $boundaryEndpoint = getenv('BOUNDARY_ENDPOINT');
    $boundaryApiKey = getenv('BOUNDARY_API_KEY');

    if ($boundaryEndpoint) {
        $config->setHost($boundaryEndpoint);
    }

    if ($boundaryApiKey) {
        $config->setAccessToken($boundaryApiKey);
    }

    $apiInstance = new OpenAPI\Client\Api\DefaultApi(
        new GuzzleHttp\Client(),
        $config
    );

    // Use `$apiInstance` to make API calls
    ```
  </Tab>

  <Tab title="Ruby">
    ```ruby
    require 'baml_client'

    api_client = BamlClient::ApiClient.new

    boundary_endpoint = ENV['BOUNDARY_ENDPOINT']
    if boundary_endpoint
      api_client.host = boundary_endpoint
    end

    boundary_api_key = ENV['BOUNDARY_API_KEY']
    if boundary_api_key
      api_client.default_headers['Authorization'] = "Bearer #{boundary_api_key}"
    end
    b = BamlClient::DefaultApi.new(api_client)
    # Use `b` to make API calls
    ```
  </Tab>

  <Tab title="Rust">
    ```rust
    let mut config = baml_client::apis::configuration::Configuration::default();
    if let Some(base_path) = std::env::var("BOUNDARY_ENDPOINT").ok() {
        config.base_path = base_path;
    }
    if let Some(api_key) = std::env::var("BOUNDARY_API_KEY").ok() {
        config.bearer_access_token = Some(api_key);
    }
    // Use `config` to make API calls
    ```
  </Tab>
</Tabs>

## Run your app with docker-compose

Replace `debug-container` with the Dockerfile for your app in the
`docker-compose.yaml` file:

```yaml
services:
  baml-over-http:
    build:
      context: .
      dockerfile: baml.Dockerfile
    networks:
      - my-app-network
    healthcheck:
      test: [ "CMD", "curl", "-f", "http://localhost:2024/_debug/ping" ]
      interval: 1s
      timeout: 100ms
      retries: 3
    ports:
      - "2024:2024"

  my-app:
    build:
      context: .
      dockerfile: my-app.Dockerfile
    depends_on:
      baml-over-http:
        condition: service_healthy
    environment:
      - BAML_ENDPOINT=http://baml-over-http:2024

  debug-container:
    image: amazonlinux:latest
    depends_on:
      baml-over-http:
        condition: service_healthy
    command: sh -c 'curl -v "$${BAML_ENDPOINT}/_debug/ping"'
    environment:
      - BAML_ENDPOINT=http://baml-over-http:2024
```

Additionally, you'll want to make sure that you generate the BAML client at
image build time, because `baml_client/` should not be checked into your repo.

This means that in the CI workflow you use to push your Docker images, you'll
want to do something like this:

```yaml .github/workflows/build-image.yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Build the BAML client
        run: |
          set -eux
          npx @boundaryml/baml generate
          docker build -t my-app .
```


# Prompting in BAML

<Note>
  We recommend reading the [installation](/guide/installation-language/python) instructions first
</Note>

BAML functions are special definitions that get converted into real code (Python, TS, etc) that calls LLMs. Think of them as a way to define AI-powered functions that are type-safe and easy to use in your application.

### What BAML Functions Actually Do

When you write a BAML function like this:

```rust BAML
function ExtractResume(resume_text: string) -> Resume {
  client "openai/gpt-4o"
  // The prompt uses Jinja syntax.. more on this soon.
  prompt #"
     Extract info from this text.

    {# special macro to print the output schema + instructions #}
    {{ ctx.output_format }}

    Resume:
    ---
    {{ resume_text }}
    ---
  "#
}
```

BAML converts it into code that:

1. Takes your input (`resume_text`)
2. Sends a request to OpenAI's GPT-4 API with your prompt.
3. Parses the JSON response into your `Resume` type
4. Returns a type-safe object you can use in your code

## Calling the function

Recall that BAML will generate a `baml_client` directory in the language of your choice using the parameters in your [`generator`](/ref/baml/generator) config. This contains the function and types you defined.

Now we can call the function, which will make a request to the LLM and return the `Resume` object:

<CodeBlocks>
  ```python python
  # Import the baml client (We call it `b` for short)
  from baml_client import b
  # Import the Resume type, which is now a Pydantic model!
  from baml_client.types import Resume 

  def main():
  resume_text = """Jason Doe\nPython, Rust\nUniversity of California, Berkeley, B.S.\nin Computer Science, 2020\nAlso an expert in Tableau, SQL, and C++\n"""

      # this function comes from the autogenerated "baml_client".
      # It calls the LLM you specified and handles the parsing.
      resume = b.ExtractResume(resume_text)

      # Fully type-checked and validated!
      assert isinstance(resume, Resume)

  ```

  ```typescript typescript
  import b from 'baml_client'
  import { Resume } from 'baml_client/types'

  async function main() {
    const resume_text = `Jason Doe\nPython, Rust\nUniversity of California, Berkeley, B.S.\nin Computer Science, 2020\nAlso an expert in Tableau, SQL, and C++`

    // this function comes from the autogenerated "baml_client".
    // It calls the LLM you specified and handles the parsing.
    const resume = await b.ExtractResume(resume_text)

    // Fully type-checked and validated!
    resume.name === 'Jason Doe'
    if (resume instanceof Resume) {
      console.log('resume is a Resume')
    }
  }
  ```

  ```ruby ruby

  require_relative "baml_client/client"
  b = Baml.Client

  # Note this is not async
  res = b.TestFnNamedArgsSingleClass(
      myArg: Baml::Types::Resume.new(
          key: "key",
          key_two: true,
          key_three: 52,
      )
  )
  ```
</CodeBlocks>

<Warning>
  Do not modify any code inside `baml_client`, as it's autogenerated.
</Warning>


# Switching LLMs

BAML Supports getting structured output from **all** major providers as well as all OpenAI-API compatible open-source models. See [LLM Providers Reference](/ref/llm-client-providers/open-ai) for how to set each one up.

<Tip>
  BAML can help you get structured output from **any Open-Source model**, with better performance than other techniques, even when it's not officially supported via a Tool-Use API (like o1-preview) or fine-tuned for it! [Read more about how BAML does this](https://www.boundaryml.com/blog/schema-aligned-parsing).
</Tip>

### Using `client "<provider>/<model>"`

Using `openai/model-name` or `anthropic/model-name` will assume you have the ANTHROPIC\_API\_KEY or OPENAI\_API\_KEY environment variables set.

```rust BAML
function MakeHaiku(topic: string) -> string {
  client "openai/gpt-4o" // or anthropic/claude-3-5-sonnet-20240620
  prompt #"
    Write a haiku about {{ topic }}.
  "#
}
```

### Using a named client

<Note>Use this if you are using open-source models or need customization</Note>
The longer form uses a named client, and supports adding any parameters supported by the provider or changing the temperature, top\_p, etc.

```rust BAML
client<llm> MyClient {
  provider "openai"
  options {
    model "gpt-4o"
    api_key env.OPENAI_API_KEY
    // other params like temperature, top_p, etc.
    temperature 0.0
    base_url "https://my-custom-endpoint.com/v1"
    // add headers
    headers {
      "anthropic-beta" "prompt-caching-2024-07-31"
    }
  }

}

function MakeHaiku(topic: string) -> string {
  client MyClient
  prompt #"
    Write a haiku about {{ topic }}.
  "#
}
```

Consult the [provider documentation](/ref/llm-client-providers/open-ai) for a list of supported providers
and models, the default options, and setting [retry policies](/ref/llm-client-strategies/retry-policy).

<Tip>
  If you want to specify which client to use at runtime, in your Python/TS/Ruby code,
  you can use the [client registry](/ref/baml-client/client-registry) to do so.

  This can come in handy if you're trying to, say, send 10% of your requests to a
  different model.
</Tip>


### Dynamic BAML Classes

Now we'll add some properties to a `User` class at runtime using @@dynamic.

```rust BAML
class User {
  name string
  age int
  @@dynamic
}

function DynamicUserCreator(user_info: string) -> User {
  client GPT4
  prompt #"
    Extract the information from this chunk of text:
    "{{ user_info }}"

    {{ ctx.output_format }}
  "#
}
```

We can then modify the `User` schema at runtime. Since we marked `User` with `@@dynamic`, it'll be available as a property of `TypeBuilder`.

<CodeBlocks>
  ```python Python
  from baml_client.type_builder import TypeBuilder
  from baml_client import b

  async def run():
    tb = TypeBuilder()
    tb.User.add_property('email', tb.string())
    tb.User.add_property('address', tb.string()).description("The user's address")
    res = await b.DynamicUserCreator("some user info", { "tb": tb })
    # Now res can have email and address fields
    print(res)

  ```

  ```typescript TypeScript
  import TypeBuilder from '../baml_client/type_builder'
  import {
    b
  } from '../baml_client'

  async function run() {
    const tb = new TypeBuilder()
    tb.User.add_property('email', tb.string())
    tb.User.add_property('address', tb.string()).description("The user's address")
    const res = await b.DynamicUserCreator("some user info", { tb: tb })
    // Now res can have email and address fields
    console.log(res)
  }
  ```

  ```ruby Ruby
  require_relative 'baml_client/client'

  def run
    tb = Baml::TypeBuilder.new
    tb.User.add_property('email', tb.string)
    tb.User.add_property('address', tb.string).description("The user's address")

    res = Baml.Client.dynamic_user_creator(input: "some user info", baml_options: {tb: tb})
    # Now res can have email and address fields
    puts res
  end
  ```
</CodeBlocks>

### Creating new dynamic classes or enums not in BAML

The previous examples showed how to modify existing types. Here we create a new `Hobbies` enum, and a new class called `Address` without having them defined in BAML.

Note that you must attach the new types to the existing Return Type of your BAML function(in this case it's `User`).

<CodeBlocks>
  ```python Python
  from baml_client.type_builder import TypeBuilder
  from baml_client.async_client import b

  async def run():
    tb = TypeBuilder()
    hobbies_enum = tb.add_enum("Hobbies")
    hobbies_enum.add_value("Soccer")
    hobbies_enum.add_value("Reading")

    address_class = tb.add_class("Address")
    address_class.add_property("street", tb.string()).description("The user's street address")

    tb.User.add_property("hobby", hobbies_enum.type().optional())
    tb.User.add_property("address", address_class.type().optional())
    res = await b.DynamicUserCreator("some user info", {"tb": tb})
    # Now res might have the hobby property, which can be Soccer or Reading
    print(res)

  ```

  ```typescript TypeScript
  import TypeBuilder from '../baml_client/type_builder'
  import { b } from '../baml_client'

  async function run() {
    const tb = new TypeBuilder()
    const hobbiesEnum = tb.addEnum('Hobbies')
    hobbiesEnum.addValue('Soccer')
    hobbiesEnum.addValue('Reading')

    const addressClass = tb.addClass('Address')
    addressClass.addProperty('street', tb.string()).description("The user's street address")


    tb.User.addProperty('hobby', hobbiesEnum.type().optional())
    tb.User.addProperty('address', addressClass.type())
    const res = await b.DynamicUserCreator("some user info", { tb: tb })
    // Now res might have the hobby property, which can be Soccer or Reading
    console.log(res)
  }
  ```

  ```ruby Ruby
  require_relative 'baml_client/client'

  def run
    tb = Baml::TypeBuilder.new
    hobbies_enum = tb.add_enum('Hobbies')
    hobbies_enum.add_value('Soccer')
    hobbies_enum.add_value('Reading')

    address_class = tb.add_class('Address')
    address_class.add_property('street', tb.string)

    tb.User.add_property('hobby', hobbies_enum.type.optional)
    tb.User.add_property('address', address_class.type.optional)

    res = Baml::Client.dynamic_user_creator(input: "some user info", baml_options: { tb: tb })
    # Now res might have the hobby property, which can be Soccer or Reading
    puts res
  end
  ```
</CodeBlocks>

TypeBuilder provides methods for building different kinds of types:

| Method       | Description              | Example                  |
| ------------ | ------------------------ | ------------------------ |
| `string()`   | Creates a string type    | `tb.string()`            |
| `int()`      | Creates an integer type  | `tb.int()`               |
| `float()`    | Creates a float type     | `tb.float()`             |
| `bool()`     | Creates a boolean type   | `tb.bool()`              |
| `list()`     | Makes a type into a list | `tb.string().list()`     |
| `optional()` | Makes a type optional    | `tb.string().optional()` |

### Adding descriptions to dynamic types

<CodeBlocks>
  ```python Python
  tb = TypeBuilder()
  tb.User.add_property("email", tb.string()).description("The user's email")
  ```

  ```typescript TypeScript
  const tb = new TypeBuilder()
  tb.User.addProperty("email", tb.string()).description("The user's email")
  ```

  ```ruby Ruby
  tb = Baml::TypeBuilder.new
  tb.User.add_property("email", tb.string).description("The user's email")
  ```
</CodeBlocks>

### Creating dynamic classes and enums at runtime with BAML

The `TypeBuilder` has a higher level API for creating dynamic types at runtime.
Here's an example:

<CodeBlocks>
  ```python Python
  tb = TypeBuilder()
  tb.add_baml("""
    // Creates a new class Address that does not exist in the BAML source.
    class Address {
      street string
      city string
      state string
    }

    // Modifies the existing @@dynamic User class to add the new address property.
    dynamic class User {
      address Address
    }

    // Modifies the existing @@dynamic Category enum to add a new variant.
    dynmic enum Category {
      VALUE5
    }
  """)
  ```

  ```typescript TypeScript
  const tb = new TypeBuilder()
  tb.addBaml(`
    // Creates a new class Address that does not exist in the BAML source.
    class Address {
      street string
      city string
      state string
    }

    // Modifies the existing @@dynamic User class to add the new address property.
    dynamic class User {
      address Address
    }

    // Modifies the existing @@dynamic Category enum to add a new variant.
    dynmic enum Category {
      VALUE5
    }
  `)
  ```

  ```ruby Ruby
  tb = Baml::TypeBuilder.new
  tb.add_baml("
    // Creates a new class Address that does not exist in the BAML source.
    class Address {
      street string
      city string
      state string
    }

    // Modifies the existing @@dynamic User class to add the new address property.
    dynamic class User {
      address Address
    }

    // Modifies the existing @@dynamic Category enum to add a new variant.
    dynmic enum Category {
      VALUE5
    }
  ")
  ```
</CodeBlocks>

### Building dynamic types from JSON schema

We have a working implementation of this, but are waiting for a concrete use case to merge it.
Please chime in on [the GitHub issue](https://github.com/BoundaryML/baml/issues/771) if this is
something you'd like to use.

<CodeBlocks>
  ```python Python
  import pydantic
  from baml_client import b

  class Person(pydantic.BaseModel):
      last_name: list[str]
      height: Optional[float] = pydantic.Field(description="Height in meters")

  tb = TypeBuilder()
  tb.unstable_features.add_json_schema(Person.model_json_schema())

  res = await b.ExtractPeople(
      "My name is Harrison. My hair is black and I'm 6 feet tall. I'm pretty good around the hoop. I like giraffes.",
      {"tb": tb},
  )
  ```

  ```typescript TypeScript
  import 'z' from zod
  import 'zodToJsonSchema' from zod-to-json-schema
  import { b } from '../baml_client'

  const personSchema = z.object({
    animalLiked: z.object({
      animal: z.string().describe('The animal mentioned, in singular form.'),
    }),
    hobbies: z.enum(['chess', 'sports', 'music', 'reading']).array(),
    height: z.union([z.string(), z.number().int()]).describe('Height in meters'),
  })

  let tb = new TypeBuilder()
  tb.unstableFeatures.addJsonSchema(zodToJsonSchema(personSchema, 'Person'))

  const res = await b.ExtractPeople(
    "My name is Harrison. My hair is black and I'm 6 feet tall. I'm pretty good around the hoop. I like giraffes.",
    { tb },
  )
  ```

  ```ruby Ruby
  tb = Baml::TypeBuilder.new
  tb.unstable_features.add_json_schema(...)

  res = Baml::Client.extract_people(
    input: "My name is Harrison. My hair is black and I'm 6 feet tall. I'm pretty good around the hoop. I like giraffes.",
    baml_options: { tb: tb }
  )

  puts res
  ```
</CodeBlocks>

# Prompt Caching / Message Role Metadata

Recall that an LLM request usually looks like this, where it sometimes has metadata in each `message`. In this case, Anthropic has a `cache_control` key.

```curl {3,11} Anthropic Request
curl https://api.anthropic.com/v1/messages \
  -H "content-type: application/json" \
  -H "anthropic-beta: prompt-caching-2024-07-31" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 1024,
    "messages": [
       {
        "type": "text", 
        "text": "<the entire contents of Pride and Prejudice>",
        "cache_control": {"type": "ephemeral"}
      },
      {
        "role": "user",
        "content": "Analyze the major themes in Pride and Prejudice."
      }
    ]
  }'
```

This is nearly the same as this BAML code, minus the `cache_control` metadata:

Let's add the `cache-control` metadata to each of our messges in BAML now.
There's just 2 steps:

<Steps>
  ### Allow role metadata and header in the client definition

  ```baml {5-8} main.baml
  client<llm> AnthropicClient {
    provider "anthropic"
    options {
      model "claude-3-5-sonnet-20241022"
      allowed_role_metadata ["cache_control"]
      headers {
        "anthropic-beta" "prompt-caching-2024-07-31"
      }
    }
  }
  ```

  ### Add the metadata to the messages

  ```baml {2,6} main.baml
  function AnalyzeBook(book: string) -> string {
    client<llm> AnthropicClient
    prompt #"
      {{ _.role("user") }}
      {{ book }}
      {{ _.role("user", cache_control={"type": "ephemeral"}) }}
      Analyze the major themes in Pride and Prejudice.
    "#
  }
  ```
</Steps>

We have the "allowed\_role\_metadata" so that if you swap to other LLM clients, we don't accidentally forward the wrong metadata to the new provider API.

<Tip>
  Remember to check the "raw curl" checkbox in the VSCode Playground to see the exact request being sent!
</Tip>



## Choosing N Tools

To choose many tools, you can use a union of a list:

```baml BAML
function UseTool(user_message: string) -> (WeatherAPI | MyOtherAPI)[] {
  client "openai/gpt-4o-mini"
  prompt #"
    Given a message, extract info.
    {# special macro to print the functions return type. #}
    {{ ctx.output_format }}

    {{ _.role('user') }}
    {{ user_message }}
  "#
}
```

Call the function like this:

<CodeGroup>
  ```python Python
  import asyncio
  from baml_client import b
  from baml_client.types import WeatherAPI, MyOtherAPI

  async def main():
      tools = b.UseTool("What's the weather like in San Francisco and New York?")
      print(tools)  
      
      for tool in tools:
          if isinstance(tool, WeatherAPI):
              print(f"Weather API called:")
              print(f"City: {tool.city}")
              print(f"Time of Day: {tool.timeOfDay}")
          elif isinstance(tool, MyOtherAPI):
              print(f"MyOtherAPI called:")
              # Handle MyOtherAPI specific attributes here

  if __name__ == '__main__':
      main()
  ```

  ```typescript TypeScript
  import { b } from './baml_client'
  import { WeatherAPI, MyOtherAPI } from './baml_client/types'

  const main = async () => {
    const tools = await b.UseTool("What's the weather like in San Francisco and New York?")
    console.log(tools)
    
    tools.forEach(tool => {
      if (tool instanceof WeatherAPI) {
        console.log("Weather API called:")
        console.log(`City: ${tool.city}`)
        console.log(`Time of Day: ${tool.timeOfDay}`)
      } else if (tool instanceof MyOtherAPI) {
        console.log("MyOtherAPI called:")
        // Handle MyOtherAPI specific attributes here
      }
    })
  }

  main()
  ```

  ```ruby Ruby
  require_relative "baml_client/client"

  $b = Baml.Client

  def main
    tools = $b.UseTool(user_message: "What's the weather like in San Francisco and New York?")
    puts tools
    
    tools.each do |tool|
      case tool
      when Baml::Types::WeatherAPI
        puts "Weather API called:"
        puts "City: #{tool.city}"
        puts "Time of Day: #{tool.timeOfDay}"
      when Baml::Types::MyOtherAPI
        puts "MyOtherAPI called:"
        # Handle MyOtherAPI specific attributes here
      end
    end
  end

  main
  ```
</CodeGroup>

## Dynamically Generate the tool signature

It might be cumbersome to define schemas in baml and code, so you can define them from code as well. Read more about dynamic types [here](/guide/baml-advanced/dynamic-runtime-types)

```baml BAML
class WeatherAPI {
  @@dynamic // params defined from code
}

function UseTool(user_message: string) -> WeatherAPI {
  client "openai/gpt-4o-mini"
  prompt #"
    Given a message, extract info.
    {# special macro to print the functions return type. #}
    {{ ctx.output_format }}

    {{ _.role('user') }}
    {{ user_message }}
  "#
}
```

Call the function like this:

<CodeGroup>
  ```python Python
  import asyncio
  import inspect

  from baml_client import b
  from baml_client.type_builder import TypeBuilder
  from baml_client.types import WeatherAPI

  async def get_weather(city: str, time_of_day: str):
      print(f"Getting weather for {city} at {time_of_day}")
      return 42

  async def main():
      tb = TypeBuilder()
      type_map = {int: tb.int(), float: tb.float(), str: tb.string()}
      signature = inspect.signature(get_weather)
      for param_name, param in signature.parameters.items():
          tb.WeatherAPI.add_property(param_name, type_map[param.annotation])
      tool = b.UseTool("What's the weather like in San Francisco this afternoon?", { "tb": tb })
      print(tool)
      weather = await get_weather(**tool.model_dump())
      print(weather)

  if __name__ == '__main__':
      asyncio.run(main())
  ```
</CodeGroup>

<Warning>
  Note that the above approach is not fully generic. Recommended you read: 

  [Dynamic JSON Schema](https://www.boundaryml.com/blog/dynamic-json-schemas)
</Warning>

## Function-calling APIs vs Prompting

Injecting your function schemas into the prompt, as BAML does, outperforms function-calling across all benchmarks for major providers ([see our Berkeley FC Benchmark results with BAML](https://www.boundaryml.com/blog/sota-function-calling?q=0)).

Amongst other limitations, function-calling APIs will at times:

1. Return a schema when you don't want any (you want an error)
2. Not work for tools with more than 100 parameters.
3. Use [many more tokens than prompting](https://www.boundaryml.com/blog/type-definition-prompting-baml).

Keep in mind that "JSON mode" is nearly the same thing as "prompting", but it enforces the LLM response is ONLY a JSON blob.
BAML does not use JSON mode since it allows developers to use better prompting techniques like chain-of-thought, to allow the LLM to express its reasoning before printing out the actual schema. BAML's parser can find the json schema(s) out of free-form text for you. Read more about different approaches to structured generation [here](https://www.boundaryml.com/blog/schema-aligned-parsing)

BAML will still support native function-calling APIs in the future (please let us know more about your use-case so we can prioritize accordingly)


# Chain-of-Thought Prompting

Chain-of-thought prompting is a technique that asdf encourages the language model to think step by step, reasoning through the problem before providing an answer. This can improve the quality of the response and make it easier to understand.

<Frame caption="Chain-of-Thought Prompting [Wei et al. (2022)](https://arxiv.org/abs/2201.11903)">
  <img src="file:e95ea226-e3ec-48cb-93e9-04298d9a4266" alt="Chain-of-Thought Prompting" />
</Frame>

There are a few different ways to implement chain-of-thought prompting, especially for structured outputs.

1. Require the model to reason before outputting the structured object.
   * Bonus: Use a `template_string` to embed the reasoning into multiple functions.
2. Require the model to **flexibly** reason before outputting the structured object.
3. Embed reasoning in the structured object.
4. Ask the model to embed reasoning as comments in the structured object.

Let's look at an example of each of these.

<Tip>
  We recommend [Technique 2](#technique-2-allowing-for-flexible-reasoning) for most use cases.
  But each technique has its own trade-offs, so please try them out and see which one works best for your use case.
</Tip>

<Info>
  Since BAML leverages [Schema-Aligned Parsing (SAP)](https://www.boundaryml.com/blog/schema-aligned-parsing) instead of JSON.parse or LLM modification (like constrained generation or structured outputs), we can do all of the above techniques with any language model!
</Info>

## Technique 1: Reasoning before outputting the structured object

In the below example, we use chain of thought prompting to extract information from an email.

```baml {9-17}
function GetOrderInfo(email: Email) -> OrderInfo {
  client "openai/gpt-4o-mini"
  prompt #"
    extract everything from this email.


    {{ ctx.output_format }}

    Before you answer, please explain your reasoning step-by-step. 
    
    For example:
    If we think step by step we can see that ...

    Therefore the output is:
    {
      ... // schema
    }

    {{ _.role('user') }}

    Sender: {{email.from_address}}
    Email Subject: {{email.subject}}
    Email Body: {{email.body}}
  "#
}

class Email {
    subject string
    body string
    from_address string
}


class OrderInfo {
    order_status "ORDERED" | "SHIPPED" | "DELIVERED" | "CANCELLED"
    tracking_number string?
    estimated_arrival_date string?
}

test Test1 {
  functions [GetOrderInfo]
  args {
    email {
      from_address "hello@amazon.com"
      subject "Your Amazon.com order of 'Wood Dowel Rods...' has shipped!"
      body #"
        Hi Sam, your package will arrive:
        Thurs, April 4
        Track your package:
        www.amazon.com/gp/your-account/ship-track?ie=23&orderId123

        On the way:
        Wood Dowel Rods...
        Order #113-7540940
        Ship to:
            Sam
            SEATTLE, WA

        Shipment total:
        $0.00
    "#

    }
  }
}
```

### Reusable Chain-of-Thought Snippets

You may want to reuse the same technique for multiple functions. Consider [template\_string](/ref/baml/template-string)!

```baml {1-12, 21}
template_string ChainOfThought(action: string?) #"
    Before you answer, please explain your reasoning step-by-step.
    {% if action %}{{ action }}{% endif %}
    
    For example:
    If we think step by step we can see that ...

    Therefore the output is:
    {
      ... // schema
    }
"#

function GetOrderInfo(email: Email) -> OrderInfo {
  client "openai/gpt-"
  prompt #"
    Extract everything from this email.

    {{ ctx.output_format }}

    {{ ChainOfThought("focus on things related to shipping") }}

    {{ _.role('user') }}

    Sender: {{email.from_address}}
    Email Subject: {{email.subject}}
    Email Body: {{email.body}}
  "#
}

```

## Technique 2: Allowing for flexible reasoning

<Tip>
  This is one we recommend for most use cases.
</Tip>

```baml {9-16}
function GetOrderInfo(email: Email) -> OrderInfo {
  client "openai/gpt-"
  prompt #"
    extract everything from this email.


    {{ ctx.output_format }}

    Outline some relevant information before you answer.
    Example:
    - ...
    - ...
    ...
    {
      ... // schema
    }

    {{ _.role('user') }}

    Sender: {{email.from_address}}
    Email Subject: {{email.subject}}
    Email Body: {{email.body}}
  "#
}
```

The benefit of using `- ...` is that we allow the model to know it needs to output some information, but we don't limit it to a specific format or inject any bias by adding example text that may not be relevant.

Similarly, we use `...` after two `- ...` to indicate that we don't mean to limit the number of items to 2.

<Accordion title="Reuseable snippet">
  ```baml {1-10, 19}
  template_string ChainOfThought() #"
      Outline some relevant information before you answer.
      Example:
      - ...
      - ...
      ...
      {
        ... // schema
      }
  "#

  function GetOrderInfo(email: Email) -> OrderInfo {
    client "openai/gpt-"
    prompt #"
      extract everything from this email.

      {{ ctx.output_format }}

      {{ ChainOfThought() }}

      {{ _.role('user') }}

      Sender: {{email.from_address}}
      Email Subject: {{email.subject}}
      Email Body: {{email.body}}
    "#
  }
  ```
</Accordion>

## Technique 3: Embed reasoning in the structured object

```baml {2-4}
class OrderInfo {
    clues string[] @description(#"
      relevant quotes from the email related to shipping
    "#)
    order_status "ORDERED" | "SHIPPED" | "DELIVERED" | "CANCELLED"
    tracking_number string?
    estimated_arrival_date string?
}

function GetOrderInfo(email: Email) -> OrderInfo {
  client "openai/gpt-"
  prompt #"
    extract everything from this email.

    {{ ctx.output_format }}

    {{ _.role('user') }}

    Sender: {{email.from_address}}
    Email Subject: {{email.subject}}
    Email Body: {{email.body}}
  "#
}
```

## Technique 4: Ask the model to embed reasoning as comments in the structured object

```baml {3-5}
class OrderInfo {
    order_status "ORDERED" | "SHIPPED" | "DELIVERED" | "CANCELLED"
      @description(#"
        before fields, in comments list out any relevant clues from the email
      "#)
    tracking_number string?
    estimated_arrival_date string?
}

function GetOrderInfo(email: Email) -> OrderInfo {
  client "openai/gpt-"
  prompt #"
    extract everything from this email.

    {{ ctx.output_format }}

    {{ _.role('user') }}

    Sender: {{email.from_address}}
    Email Subject: {{email.subject}}
    Email Body: {{email.body}}
  "#
}
```


# TypeBuilder

`TypeBuilder` is used to create or modify output schemas at runtime. It's particularly useful when you have dynamic output structures that can't be determined at compile time - like categories from a database or user-provided schemas.

Here's a simple example of using TypeBuilder to add new enum values before calling a BAML function:

**BAML Code**

```baml {4}
enum Category {
  RED
  BLUE
  @@dynamic  // Makes this enum modifiable at runtime
}

function Categorize(text: string) -> Category {
  prompt #"
    Categorize this text:
    {{ text }}

    {{ ctx.output_format }}
  "#
}
```

**Runtime Usage**

<CodeBlocks>
  ```python Python
  from baml_client.type_builder import TypeBuilder
  from baml_client import b

  # Create a TypeBuilder instance
  tb = TypeBuilder()

  # Add new values to the Category enum
  tb.Category.add_value('GREEN')
  tb.Category.add_value('YELLOW')

  # Pass the typebuilder when calling the function
  result = b.Categorize("The sun is bright", {"tb": tb})
  # result can now be RED, BLUE, GREEN, or YELLOW
  ```

  ```typescript TypeScript
  import { TypeBuilder } from '../baml_client/type_builder'
  import { b } from '../baml_client'

  // Create a TypeBuilder instance
  const tb = new TypeBuilder()

  // Add new values to the Category enum
  tb.Category.addValue('GREEN')
  tb.Category.addValue('YELLOW')

  // Pass the typebuilder when calling the function
  const result = await b.Categorize("The sun is bright", { tb })
  // result can now be RED, BLUE, GREEN, or YELLOW
  ```

  ```ruby Ruby
  require_relative 'baml_client/client'

  # Create a TypeBuilder instance
  tb = Baml::TypeBuilder.new

  # Add new values to the Category enum
  tb.Category.add_value('GREEN')
  tb.Category.add_value('YELLOW')

  # Pass the typebuilder when calling the function
  result = Baml::Client.categorize(text: "The sun is bright", baml_options: { tb: tb })
  # result can now be RED, BLUE, GREEN, or YELLOW
  ```
</CodeBlocks>

## Dynamic Types

There are two ways to use TypeBuilder:

1. Modifying existing BAML types marked with `@@dynamic`
2. Creating entirely new types at runtime

### Modifying Existing Types

To modify an existing BAML type, mark it with `@@dynamic`:

<ParamField path="Classes" type="example">
  ```baml
  class User {
    name string
    age int
    @@dynamic  // Allow adding more properties
  }
  ```

  **Runtime Usage**

  <CodeBlocks>
    ```python Python
    tb = TypeBuilder()
    tb.User.add_property('email', tb.string())
    tb.User.add_property('address', tb.string())
    ```

    ```typescript TypeScript
    const tb = new TypeBuilder()
    tb.User.addProperty('email', tb.string())
    tb.User.addProperty('address', tb.string())
    ```

    ```ruby Ruby
    tb = Baml::TypeBuilder.new
    tb.User.add_property('email', tb.string)
    tb.User.add_property('address', tb.string)
    ```
  </CodeBlocks>
</ParamField>

<ParamField path="Enums" type="example">
  ```baml
  enum Category {
    VALUE1
    VALUE2
    @@dynamic  // Allow adding more values
  }
  ```

  **Runtime Usage**

  <CodeBlocks>
    ```python Python
    tb = TypeBuilder()
    tb.Category.add_value('VALUE3')
    tb.Category.add_value('VALUE4')
    ```

    ```typescript TypeScript
    const tb = new TypeBuilder()
    tb.Category.addValue('VALUE3')
    tb.Category.addValue('VALUE4')
    ```

    ```ruby Ruby
    tb = Baml::TypeBuilder.new
    tb.Category.add_value('VALUE3')
    tb.Category.add_value('VALUE4')
    ```
  </CodeBlocks>
</ParamField>

### Creating New Types

You can also create entirely new types at runtime:

<CodeBlocks>
  ```python Python
  tb = TypeBuilder()

  # Create a new enum
  hobbies = tb.add_enum("Hobbies")
  hobbies.add_value("Soccer")
  hobbies.add_value("Reading")

  # Create a new class
  address = tb.add_class("Address")
  address.add_property("street", tb.string())
  address.add_property("city", tb.string())

  # Attach new types to existing BAML type
  tb.User.add_property("hobbies", hobbies.type().list())
  tb.User.add_property("address", address.type())
  ```

  ```typescript TypeScript
  const tb = new TypeBuilder()

  // Create a new enum
  const hobbies = tb.addEnum("Hobbies")
  hobbies.addValue("Soccer")
  hobbies.addValue("Reading")

  // Create a new class
  const address = tb.addClass("Address")
  address.addProperty("street", tb.string())
  address.addProperty("city", tb.string())

  // Attach new types to existing BAML type
  tb.User.addProperty("hobbies", hobbies.type().list())
  tb.User.addProperty("address", address.type())
  ```

  ```ruby Ruby
  tb = Baml::TypeBuilder.new

  # Create a new enum
  hobbies = tb.add_enum("Hobbies")
  hobbies.add_value("Soccer")
  hobbies.add_value("Reading")

  # Create a new class
  address = tb.add_class("Address")
  address.add_property("street", tb.string)
  address.add_property("city", tb.string)

  # Attach new types to existing BAML type
  tb.User.add_property("hobbies", hobbies.type.list)
  tb.User.add_property("address", address.type)
  ```
</CodeBlocks>

## Type Builders

TypeBuilder provides methods for building different kinds of types:

| Method                      | Returns        | Description              | Example                             |
| --------------------------- | -------------- | ------------------------ | ----------------------------------- |
| `string()`                  | `FieldType`    | Creates a string type    | `tb.string()`                       |
| `int()`                     | `FieldType`    | Creates an integer type  | `tb.int()`                          |
| `float()`                   | `FieldType`    | Creates a float type     | `tb.float()`                        |
| `bool()`                    | `FieldType`    | Creates a boolean type   | `tb.bool()`                         |
| `list(type: FieldType)`     | `FieldType`    | Makes a type into a list | `tb.list(tb.string())`              |
| `union(types: FieldType[])` | `FieldType`    | Creates a union of types | `tb.union([tb.string(), tb.int()])` |
| `add_class(name: string)`   | `ClassBuilder` | Creates a new class      | `tb.add_class("User")`              |
| `add_enum(name: string)`    | `EnumBuilder`  | Creates a new enum       | `tb.add_enum("Category")`           |

In addition to the methods above, all types marked with `@@dynamic` will also appear in the TypeBuilder.

```baml {4}
class User {
  name string
  age int
  @@dynamic  // Allow adding more properties
}
```

```python {2}
tb = TypeBuilder()
tb.User.add_property("email", tb.string())
```

### FieldType

`FieldType` is a type that represents a field in a type. It can be used to add descriptions, constraints, and other metadata to a field.

| Method       | Returns     | Description              | Example                  |
| ------------ | ----------- | ------------------------ | ------------------------ |
| `list()`     | `FieldType` | Makes a type into a list | `tb.string().list()`     |
| `optional()` | `FieldType` | Makes a type optional    | `tb.string().optional()` |

### ClassBuilder

`ClassBuilder` is a type that represents a class in a type. It can be used to add properties to a class.

| Method                                        | Returns                | Description                     | Example                                     |
| --------------------------------------------- | ---------------------- | ------------------------------- | ------------------------------------------- |
| `add_property(name: string, type: FieldType)` | `ClassPropertyBuilder` | Adds a property to the class    | `my_cls.add_property("email", tb.string())` |
| `description(description: string)`            | `ClassBuilder`         | Adds a description to the class | `my_cls.description("A user class")`        |
| `type()`                                      | `FieldType`            | Returns the type of the class   | `my_cls.type()`                             |

### ClassPropertyBuilder

`ClassPropertyBuilder` is a type that represents a property in a class. It can be used to add descriptions, constraints, and other metadata to a property.

| Method                             | Returns                | Description                              | Example                                   |
| ---------------------------------- | ---------------------- | ---------------------------------------- | ----------------------------------------- |
| `description(description: string)` | `ClassPropertyBuilder` | Adds a description to the property       | `my_prop.description("An email address")` |
| `alias(alias: string)`             | `ClassPropertyBuilder` | Adds the alias attribute to the property | `my_prop.alias("email_address")`          |

### EnumBuilder

`EnumBuilder` is a type that represents an enum in a type. It can be used to add values to an enum.

| Method                             | Returns            | Description                          | Example                                      |
| ---------------------------------- | ------------------ | ------------------------------------ | -------------------------------------------- |
| `add_value(value: string)`         | `EnumValueBuilder` | Adds a value to the enum             | `my_enum.add_value("VALUE1")`                |
| `description(description: string)` | `EnumBuilder`      | Adds a description to the enum value | `my_enum.description("A value in the enum")` |
| `type()`                           | `FieldType`        | Returns the type of the enum         | `my_enum.type()`                             |

### EnumValueBuilder

`EnumValueBuilder` is a type that represents a value in an enum. It can be used to add descriptions, constraints, and other metadata to a value.

| Method                             | Returns            | Description                                | Example                                       |
| ---------------------------------- | ------------------ | ------------------------------------------ | --------------------------------------------- |
| `description(description: string)` | `EnumValueBuilder` | Adds a description to the enum value       | `my_value.description("A value in the enum")` |
| `alias(alias: string)`             | `EnumValueBuilder` | Adds the alias attribute to the enum value | `my_value.alias("VALUE1")`                    |
| `skip()`                           | `EnumValueBuilder` | Adds the skip attribute to the enum value  | `my_value.skip()`                             |

## Adding Descriptions

You can add descriptions to properties and enum values to help guide the LLM:

<CodeBlocks>
  ```python Python
  tb = TypeBuilder()

  # Add description to a property
  tb.User.add_property("email", tb.string()) \
     .description("User's primary email address")

  # Add description to an enum value
  tb.Category.add_value("URGENT") \
     .description("Needs immediate attention")
  ```

  ```typescript TypeScript
  const tb = new TypeBuilder()

  // Add description to a property
  tb.User.addProperty("email", tb.string())
     .description("User's primary email address")

  // Add description to an enum value
  tb.Category.addValue("URGENT")
     .description("Needs immediate attention")
  ```

  ```ruby Ruby
  tb = Baml::TypeBuilder.new

  # Add description to a property
  tb.User.add_property("email", tb.string)
     .description("User's primary email address")

  # Add description to an enum value
  tb.Category.add_value("URGENT")
     .description("Needs immediate attention")
  ```
</CodeBlocks>

## Creating/modyfing dynamic types with the `add_baml` method

The `TypeBuilder` has a higher level API for creating dynamic types at runtime.
Here's an example:

<CodeBlocks>
  ```python Python
  tb = TypeBuilder()
  tb.add_baml("""
    // Creates a new class Address that does not exist in the BAML source.
    class Address {
      street string
      city string
      state string
    }

    // Modifies the existing @@dynamic User class to add the new address property.
    dynamic class User {
      address Address
    }

    // Modifies the existing @@dynamic Category enum to add a new variant.
    dynmic enum Category {
      PURPLE
    }
  """)
  ```

  ```typescript TypeScript
  const tb = new TypeBuilder()
  tb.addBaml(`
    // Creates a new class Address that does not exist in the BAML source.
    class Address {
      street string
      city string
      state string
    }

    // Modifies the existing @@dynamic User class to add the new address property.
    dynamic class User {
      address Address
    }

    // Modifies the existing @@dynamic Category enum to add a new variant.
    dynmic enum Category {
      PURPLE
    }
  `)
  ```

  ```ruby Ruby
  tb = Baml::TypeBuilder.new
  tb.add_baml("
    // Creates a new class Address that does not exist in the BAML source.
    class Address {
      street string
      city string
      state string
    }

    // Modifies the existing @@dynamic User class to add the new address property.
    dynamic class User {
      address Address
    }

    // Modifies the existing @@dynamic Category enum to add a new variant.
    dynmic enum Category {
      PURPLE
    }
  ")
  ```
</CodeBlocks>

## Common Patterns

Here are some common patterns when using TypeBuilder:

1. **Dynamic Categories**: When categories come from a database or external source

<CodeBlocks>
  ```python Python
  categories = fetch_categories_from_db()
  tb = TypeBuilder()
  for category in categories:
      tb.Category.add_value(category)
  ```

  ```typescript TypeScript
  const categories = await fetchCategoriesFromDb()
  const tb = new TypeBuilder()
  categories.forEach(category => {
      tb.Category.addValue(category)
  })
  ```

  ```ruby Ruby
  categories = fetch_categories_from_db
  tb = Baml::TypeBuilder.new
  categories.each do |category|
      tb.Category.add_value(category)
  end
  ```
</CodeBlocks>

2. **Form Fields**: When extracting dynamic form fields

<CodeBlocks>
  ```python Python
  fields = get_form_fields()
  tb = TypeBuilder()
  form = tb.add_class("Form")
  for field in fields:
      form.add_property(field.name, tb.string())
  ```

  ```typescript TypeScript
  const fields = getFormFields()
  const tb = new TypeBuilder()
  const form = tb.addClass("Form")
  fields.forEach(field => {
      form.addProperty(field.name, tb.string())
  })
  ```

  ```ruby Ruby
  fields = get_form_fields
  tb = Baml::TypeBuilder.new
  form = tb.add_class("Form")
  fields.each do |field|
      form.add_property(field.name, tb.string)
  end
  ```
</CodeBlocks>

3. **Optional Properties**: When some fields might not be present

<CodeBlocks>
  ```python Python
  tb = TypeBuilder()
  tb.User.add_property("middle_name", tb.string().optional())
  ```

  ```typescript TypeScript
  const tb = new TypeBuilder()
  tb.User.addProperty("middle_name", tb.string().optional())
  ```

  ```ruby Ruby
  tb = Baml::TypeBuilder.new
  tb.User.add_property("middle_name", tb.string.optional)
  ```
</CodeBlocks>

<Warning>
  All types added through TypeBuilder must be connected to the return type of your BAML function. Standalone types that aren't referenced won't affect the output schema.
</Warning>

## Testing Dynamic Types

See the [advanced dynamic types tests guide](/guide/baml-advanced/dynamic-runtime-types#testing-dynamic-types-in-baml)
for examples of testing functions that use dynamic types. See also the
[reference](/ref/baml/test) for syntax.

## Future Features

We're working on additional features for TypeBuilder:

* JSON Schema support (awaiting use cases)
* OpenAPI schema integration
* Pydantic model support

If you're interested in these features, please join the discussion in our GitHub
issues.



# What is Jinja / Cookbook

BAML Prompt strings are essentially [Minijinja](https://docs.rs/minijinja/latest/minijinja/filters/index.html#functions) templates, which offer the ability to express logic and data manipulation within strings. Jinja is a very popular and mature templating language amongst Python developers, so Github Copilot or another LLM can already help you write most of the logic you want.

## Jinja Cookbook

When in doubt -- use the BAML VSCode Playground preview. It will show you the fully rendered prompt, even when it has complex logic.

### Basic Syntax

* `{% ... %}`: Use for executing statements such as for-loops or conditionals.
* `{{ ... }}`: Use for outputting expressions or variables.
* `{# ... #}`: Use for comments within the template, which will not be rendered.

### Loops / Iterating Over Lists

Here's how you can iterate over a list of items, accessing each item's attributes:

```jinja Jinja
function MyFunc(messages: Message[]) -> string {
  prompt #"
    {% for message in messages %}
      {{ message.user_name }}: {{ message.content }}
    {% endfor %}
  "#
}
```

### Conditional Statements

Use conditional statements to control the flow and output of your templates based on conditions:

```jinja Jinja
function MyFunc(user: User) -> string {
  prompt #"
    {% if user.is_active %}
      Welcome back, {{ user.name }}!
    {% else %}
      Please activate your account.
    {% endif %}
  "#
}
```

### Setting Variables

You can define and use variables within your templates to simplify expressions or manage data:

```jinja
function MyFunc(items: Item[]) -> string {
  prompt #"
    {% set total_price = 0 %}
    {% for item in items %}
      {% set total_price = total_price + item.price %}
    {% endfor %}
    Total price: {{ total_price }}
  "#
}
```

### Including other Templates

To promote reusability, you can include other templates within a template. See [template strings](/ref/baml/template-string):

```baml
template_string PrintUserInfo(arg1: string, arg2: User) #"
  {{ arg1 }}
  The user's name is: {{ arg2.name }}
"#

function MyFunc(arg1: string, user: User) -> string {
  prompt #"
    Here is the user info:
    {{ PrintUserInfo(arg1, user) }}
  "#
}
```

### Built-in filters

See [jinja docs](https://jinja.palletsprojects.com/en/3.1.x/templates/#list-of-builtin-filters)


# ctx.output_format

`{{ ctx.output_format }}` is used within a prompt template (or in any template\_string) to print out the function's output schema into the prompt. It describes to the LLM how to generate a structure BAML can parse (usually JSON).

Here's an example of a function with `{{ ctx.output_format }}`, and how it gets rendered by BAML before sending it to the LLM.

**BAML Prompt**

```baml
class Resume {
  name string
  education Education[]
}
function ExtractResume(resume_text: string) -> Resume {
  prompt #"
    Extract this resume:
    ---
    {{ resume_text }}
    ---

    {{ ctx.output_format }}
  "#
}
```

**Rendered prompt**

```text
Extract this resume
---
Aaron V.
Bachelors CS, 2015
UT Austin
---

Answer in JSON using this schema: 
{
  name: string
  education: [
    {
      school: string
      graduation_year: string
    }
  ]
}
```

## Controlling the output\_format

`ctx.output_format` can also be called as a function with parameters to customize how the schema is printed, like this:

```text

{{ ctx.output_format(prefix="If you use this schema correctly and I'll tip $400:\n", always_hoist_enums=true)}}
```

Here's the parameters:

<ParamField path="prefix" type="string">
  The prefix instruction to use before printing out the schema.

  ```text
  Answer in this schema correctly I'll tip $400:
  {
    ...
  }
  ```

  BAML's default prefix varies based on the function's return type.

  | Fuction return type | Default Prefix                                  |
  | ------------------- | ----------------------------------------------- |
  | Primitive (String)  |                                                 |
  | Primitive (Int)     | `Answer as an `                                 |
  | Primitive (Other)   | `Answer as a `                                  |
  | Enum                | `Answer with any of the categories:\n`          |
  | Class               | `Answer in JSON using this schema:\n`           |
  | List                | `Answer with a JSON Array using this schema:\n` |
  | Union               | `Answer in JSON using any of these schemas:\n`  |
  | Optional            | `Answer in JSON using this schema:\n`           |
</ParamField>

<ParamField path="always_hoist_enums" type="boolean">
  Whether to inline the enum definitions in the schema, or print them above. **Default: false**

  **Inlined**

  ```

  Answer in this json schema:
  {
    categories: "ONE" | "TWO" | "THREE"
  }
  ```

  **hoisted**

  ```
  MyCategory
  ---
  ONE
  TWO
  THREE

  Answer in this json schema:
  {
    categories: MyCategory
  }
  ```

  <Warning>
    BAML will always hoist if you add a 

    [description](/docs/snippets/enum#aliases-descriptions)

     to any of the enum values.
  </Warning>
</ParamField>

<ParamField path="or_splitter" type="string">
  **Default: `or`**

  If a type is a union like `string | int` or an optional like `string?`, this indicates how it's rendered.

  BAML renders it as `property: string or null` as we have observed some LLMs have trouble identifying what `property: string | null` means (and are better with plain english).

  You can always set it to `|` or something else for a specific model you use.
</ParamField>

<ParamField path="hoisted_class_prefix" type="string">
  Prefix of hoisted classes in the prompt. **Default: `<none>`**

  Recursive classes are hoisted in the prompt so that any class field can
  reference them using their name. This parameter controls the prefix used for
  hoisted classes as well as the word used in the render message to refer to the
  output type, which defaults to `"schema"`:

  ```
  Answer in JSON using this schema:
  ```

  See examples below.

  **Recursive BAML Prompt Example**

  ```baml
  class Node {
    data int
    next Node?
  }

  class LinkedList {
    head Node?
    len int
  }

  function BuildLinkedList(input: int[]) -> LinkedList {
    prompt #"
      Build a linked list from the input array of integers.

      INPUT: {{ input }}

      {{ ctx.output_format }}    
    "#
  }
  ```

  **Default `hoisted_class_prefix` (none)**

  ```
  Node {
    data: int,
    next: Node or null
  }

  Answer in JSON using this schema:
  {
    head: Node or null,
    len: int
  }
  ```

  **Custom Prefix: `hoisted_class_prefix="interface"`**

  ```
  interface Node {
    data: int,
    next: Node or null
  }

  Answer in JSON using this interface:
  {
    head: Node or null,
    len: int
  }
  ```
</ParamField>

## Why BAML doesn't use JSON schema format in prompts

BAML uses "type definitions" or "jsonish" format instead of the long-winded json-schema format.
The tl;dr is that json schemas are

1. 4x more inefficient than "type definitions".
2. very unreadable by humans (and hence models)
3. perform worse than type definitions (especially on deeper nested objects or smaller models)

Read our [full article on json schema vs type definitions](https://www.boundaryml.com/blog/type-definition-prompting-baml)


# ctx (accessing metadata)

If you try rendering `{{ ctx }}` into the prompt (literally just write that out!), you'll see all the metadata we inject to run this prompt within the playground preview.

In the earlier tutorial we mentioned `ctx.output_format`, which contains the schema, but you can also access client information:

## Usecase: Conditionally render based on client provider

In this example, we render the list of messages in XML tags if the provider is Anthropic (as they recommend using them as delimiters). See also  [template\_string](/ref/baml/template-string) as it's used in here.

```baml
template_string RenderConditionally(messages: Message[]) #"
  {% for message in messages %}
    {%if ctx.client.provider == "anthropic" %}
      <Message>{{ message.user_name }}: {{ message.content }}</Message>
    {% else %}
      {{ message.user_name }}: {{ message.content }}
    {% endif %}
  {% endfor %}
"#

function MyFuncWithGPT4(messages: Message[]) -> string {
  client GPT4o
  prompt #"
    {{ RenderConditionally(messages)}}
  "#
}

function MyFuncWithAnthropic(messages: Message[]) -> string {
  client Claude35
  prompt #"
    {{ RenderConditionally(messages )}}
  #"
}
```


# _.role

BAML prompts are compiled into a `messages` array (or equivalent) that most LLM providers use:

BAML Prompt -> `[{ role: "user": content: "hi there"}, { role: "assistant", ...}]`

By default, BAML puts everything into a single message with the `system` role if available (or whichever one is best for the provider you have selected).
When in doubt, the playground always shows you the current role for each message.

To specify a role explicitly, add the `{{ _.role("user")}}` syntax to the prompt

```rust
prompt #"
  {{ _.role("system") }} Everything after
  this element will be a system prompt!

  {{ _.role("user")}} 
  And everything after this
  will be a user role
"#
```

Try it out in [PromptFiddle](https://www.promptfiddle.com)

<Note>
  BAML may change the default role to `user` if using specific APIs that only support user prompts, like when using prompts with images.
</Note>

We use `_` as the prefix of `_.role()` since we plan on adding more helpers here in the future.

## Example -- Using `_.role()` in for-loops

Here's how you can inject a list of user/assistant messages and mark each as a user or assistant role:

```rust BAML
class Message {
  role string
  message string
}

function ChatWithAgent(input: Message[]) -> string {
  client GPT4o
  prompt #"
    {% for m in messages %}
      {{ _.role(m.role) }}
      {{ m.message }}
    {% endfor %}
  "#
}
```

```rust BAML
function ChatMessages(messages: string[]) -> string {
  client GPT4o
  prompt #"
    {% for m in messages %}
      {{ _.role("user" if loop.index % 2 == 1 else "assistant") }}
      {{ m }}
    {% endfor %}
  "#
}
```

## Example -- Using `_.role()` in a template string

```baml BAML
template_string YouAreA(name: string, job: string) #"
  {{ _.role("system") }} 
  You are an expert {{ name }}. {{ job }}

  {{ ctx.output_format }}
  {{ _.role("user") }}
"#

function CheckJobPosting(post: string) -> bool {
  client GPT4o
  prompt #"
    {{ YouAreA("hr admin", "You're role is to ensure every job posting is bias free.") }}

    {{ post }}
  "#
}
```


# Variables

See [template\_string](/ref/baml/template-string) to learn how to add variables in .baml files


# Conditionals

Use conditional statements to control the flow and output of your templates based on conditions:

```jinja
function MyFunc(user: User) -> string {
  prompt #"
    {% if user.is_active %}
      Welcome back, {{ user.name }}!
    {% else %}
      Please activate your account.
    {% endif %}
  "#
}
```


# Loops

Here's how you can iterate over a list of items, accessing each item's attributes:

```jinja
function MyFunc(messages: Message[]) -> string {
  prompt #"
    {% for message in messages %}
      {{ message.user_name }}: {{ message.content }}
    {% endfor %}
  "#
}
```
