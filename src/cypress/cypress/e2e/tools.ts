
export enum Pages {
    InstalledApps = "installed-apps",
    AppStore = "store",
    Users = "users",
    Backups = "backups",
    Settings = "settings",
}

export function goTo(page: Pages): void {
    cy.get(`#go-to-${page}`).click()
    if (page === Pages.InstalledApps) {
        cy.location('pathname').should('equal', '/')
    } else {
        cy.location('pathname').should('equal', `/${page}`)
    }
}

export function clickConfirmationButton() {
    cy.get("#button-confirmation").click()
    cy.get("#button-confirmation").should("not.exist")
}