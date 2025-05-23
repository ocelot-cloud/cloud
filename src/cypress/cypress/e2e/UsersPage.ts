import {clickConfirmationButton, goTo, Pages} from "./tools";

export class UsersPage {
    constructor() {
        goTo(Pages.Users)
    }

    assertAdminUser() {
        cy.get('#user-list').find('tbody tr').should('have.length.gte', 1);
        cy.get('#user-list')
            .find('tr')
            .find('.name-column')
            .contains('admin')
            .parent('tr')
            .within(() => {
                cy.get('.role-column').should('contain.text', 'admin');
                cy.get('#delete-user-button')
                    .should('have.attr', 'disabled');
            });
        return this
    }

    createUser() {
        cy.get('#username').type('user')
        cy.get('#password').type('password')
        cy.get('#submit').click()

        cy.get('#user-list').find('tbody tr').should('have.length', 1);
        cy.get('#user-list').find('tr')
            .find('.name-column')
            .contains('user')
            .parent('tr')
            .within(() => {
                cy.get('.role-column').should('contain.text', 'user');
                cy.get('#delete-user-button')
                    .should('not.have.attr', 'disabled');
            });
        return this
    }

    deleteUser() {
        cy.get('#user-list .name-column')
            .contains('user')
            .closest('tr')
            .find('#delete-user-button')
            .click();
        clickConfirmationButton()
        return this
    }

    deleteUserIfExists() {
        cy.reload()
        cy.get('body').then(($body) => {
            if ($body.find('#user-list .name-column:contains("user")').length > 0) {
                this.deleteUser()
            }
        });
        return this;
    }

    shouldUserExist(shouldExist: boolean) {
        let prefix = shouldExist ? '' : 'not.'
        cy.get('body').should(prefix +'contain.text', 'user')
        return this
    }

    shouldUserNameInputBeEmpty(shouldBeEmpty: boolean) {
        let prefix = shouldBeEmpty ? '' : 'not.'
        cy.get('#username').invoke('val').should(prefix+'be.empty')
        return this
    }

    shouldUserPasswordInputBeEmpty(shouldBeEmpty: boolean) {
        let prefix = shouldBeEmpty ? '' : 'not.'
        cy.get('#password').invoke('val').should(prefix+'be.empty')
        return this
    }
}