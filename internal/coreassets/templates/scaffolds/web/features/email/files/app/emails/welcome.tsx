import { Body, Button, Container, Head, Heading, Html, Text } from "react-email";

export interface WelcomeEmailProps {
    name: string;
    loginUrl: string;
}

/**
 * A React Email template. Preview it with `pnpm exec email dev` (renders every
 * template under app/emails/), and send it with the helper in app/server/send-email.tsx.
 */
export function WelcomeEmail({ loginUrl, name }: WelcomeEmailProps) {
    return (
        <Html>
            <Head />
            <Body style={{ backgroundColor: "#ffffff", fontFamily: "sans-serif" }}>
                <Container>
                    <Heading>Welcome, {name}!</Heading>
                    <Text>Thanks for signing up. Click below to get started.</Text>
                    <Button href={loginUrl}>Open the app</Button>
                </Container>
            </Body>
        </Html>
    );
}
