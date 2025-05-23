import {clickConfirmationButton, goTo, Pages} from "./tools";

let sampleHost = "localhost"
let sampleSshPort = "2222"
let sampleSshUser = "sshadmin"
let sampleSshPassword = "ssh-password"
let sampleEncryptionPassword = "restic-password"

export class SettingsPage {
    constructor() {
        goTo(Pages.Settings)
    }

    assertHostValue(host: string) {
        cy.get('#host-text-field').should('have.value', host)
        return this
    }

    setAndSaveHostValue(host: string) {
        cy.get('#host-text-field').clear()
        cy.get('#host-text-field').type(host)
        cy.get('#save-button').click()
        return this
    }

    shouldRemoteBackupsBeEnabled(isEnabled: boolean) {
        let remoteBackupEnabledStatus: String
        if (isEnabled) {
            remoteBackupEnabledStatus = "mdi-checkbox-marked"
        } else {
            remoteBackupEnabledStatus = "mdi-checkbox-blank-outline"
        }
        cy.get("#remote-backup-checkbox")
            .closest(".v-selection-control__wrapper")
            .find(".v-selection-control__input i")
            .should("have.class", remoteBackupEnabledStatus);
        return this
    }

    assertEmptySshSettings() {
        cy.get("#remote-backup-host").should("have.value", "");
        cy.get("#remote-backup-ssh-port").should("have.value", "")
        cy.get("#remote-backup-ssh-user").should("have.value", "")
        cy.get("#remote-backup-ssh-password").should("have.value", "")
        cy.get("#remote-backup-encryption-password").should("have.value", "")
        return this
    }

    enableRemoteBackups() {
        cy.wait(500)
        cy.get("#remote-backup-checkbox").click();
        return this
    }

    addSampleSshSettings() {
        cy.get("#remote-backup-host").type(sampleHost)
        cy.get("#remote-backup-ssh-port").type(sampleSshPort)
        cy.get("#remote-backup-ssh-user").type(sampleSshUser)
        cy.get("#remote-backup-ssh-password").type(sampleSshPassword)
        cy.get("#remote-backup-encryption-password").type(sampleEncryptionPassword)
        return this
    }

    saveRemoteBackupsSettings() {
        cy.get("#remote-backup-save-button").click()
        clickConfirmationButton()
        return this
    }

    readAndAssertSshSettings() {
        cy.get("#remote-backup-host").should('have.value', sampleHost);
        cy.get("#remote-backup-ssh-port").should('have.value', sampleSshPort);
        cy.get("#remote-backup-ssh-user").should('have.value', sampleSshUser);
        cy.get("#remote-backup-ssh-password").should('have.value', sampleSshPassword);
        cy.get("#remote-backup-encryption-password").should('have.value', sampleEncryptionPassword);
        return this
    }

    assertSshKnownHostsAndTestConnection() {
        cy.get("#remote-backup-ssh-known-hosts").should("have.value", "")
        cy.window().then((win) => {
            cy.stub(win, 'alert').as('alert')
            cy.get("#remote-backup-get-known-hosts").click()
            cy.get('@alert').should('have.been.calledWith', 'received known hosts successfully')

            cy.get("#remote-backup-ssh-known-hosts").invoke("val").should("include", "[localhost]:2222")
            this.saveRemoteBackupsSettings()

            cy.get("#remote-backup-test-access").click()
            cy.get('@alert').should('have.been.calledWith', 'SSH access test was successful')
        })

        return this
    }

    assertEnableAutoUpdates(shouldBeEnabled: boolean) {
        if (shouldBeEnabled) {
            cy.get('#auto-updates-enabled-checkbox').should('be.checked')
        } else {
            cy.get('#auto-updates-enabled-checkbox').should('not.be.checked')
        }
        return this
    }

    assertEnableAutoBackups(shouldBeEnabled: boolean) {
        if (shouldBeEnabled) {
            cy.get('#auto-backups-enabled-checkbox').should('be.checked')
        } else {
            cy.get('#auto-backups-enabled-checkbox').should('not.be.checked')
        }
        return this
    }

    assertMaintenanceHour(number: number) {
        let expectedTime = number.toString() + ":00"
        cy.get('body').should('contain.text', expectedTime)
        return this
    }

    setCustomMaintenanceSettings() {
        cy.get('#auto-updates-enabled-checkbox').click()
        cy.get('#auto-backups-enabled-checkbox').click()
        cy.get('#preferred-hour-selection-dropdown').click({force: true})
        cy.get('.v-list-item-title').contains('7:00').click()
        cy.get('#maintenance-save-button').click()
        return this
    }

    doDnsChallenge() {
        cy.get("body").should("not.contain.text", "_acme-challenge")
        cy.get("#certificate-generation-email-field").type("sample@sample.com")
        cy.get("#start-dns-01-challenge-button").click()
        cy.get("body").should("contain.text", "_acme-challenge.sample.com")
        cy.get("#dns-01-challenge-record-value").should("have.value", "CaxCTxqOWo7FQjoRdRgxRoriPnOrp8PeMbCdPnh2Y84\nuJi0naVfLcobf2bK_t4-VkS0HFCK0U1WXkpGCGl3irE")
        // TODO check copy icon?
        return this
    }
}